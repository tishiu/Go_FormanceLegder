package projector

import (
	"Go_FormanceLegder/internal/events"
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Projector struct {
	DB         *pgxpool.Pool
	EventStore events.EventBoundary
}

func NewProjector(db *pgxpool.Pool) *Projector {
	return &Projector{
		DB:         db,
		EventStore: events.NewPGXStore(),
	}
}

func (p *Projector) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := p.projectBatch(ctx); err != nil {
				log.Printf("projection error: %v", err)
			}
		}
	}
}

func (p *Projector) projectBatch(ctx context.Context) error {
	tx, err := p.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var lastProcessedEventID string
	err = tx.QueryRow(ctx, `
		SELECT last_processed_event_id
		FROM projector_offsets
		WHERE projector_name = 'ledger'
	`).Scan(&lastProcessedEventID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		lastProcessedEventID = ""
	}

	eventBatch, maxEventID, err := p.eventStore().LoadTransactionPosted(ctx, tx, lastProcessedEventID, 100)
	if err != nil {
		return err
	}

	if len(eventBatch) == 0 {
		return tx.Commit(ctx)
	}

	// Process
	for _, event := range eventBatch {
		if err := p.applyTransactionPosted(ctx, tx, event.LedgerID, event.Data); err != nil {
			return fmt.Errorf("failed apply event %s: %w", event.EventID, err)
		}
	}

	// Update Offset
	_, err = tx.Exec(ctx, `
       INSERT INTO projector_offsets (projector_name, last_processed_event_id)
       VALUES ('ledger', $1)
       ON CONFLICT (projector_name)
       DO UPDATE SET last_processed_event_id = EXCLUDED.last_processed_event_id
    `, maxEventID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (p *Projector) applyTransactionPosted(ctx context.Context, tx pgx.Tx, ledgerID string, payload events.TransactionPosted) error {
	occurredAt, err := time.Parse(time.RFC3339Nano, payload.OccurredAtRFC3339Nano)
	if err != nil {
		return fmt.Errorf("invalid time format: %w", err)
	}

	// Insert transaction
	// tag.RowsAffected() == 1: Insert successful
	// tag.RowsAffected() == 0: (Old Transaction) -> RETURN
	tag, err := tx.Exec(ctx, `
       INSERT INTO transactions (
          id, ledger_id, external_id, amount, currency, occurred_at
       ) VALUES ($1, $2, $3, $4, $5, $6)
       ON CONFLICT (id, ledger_id) DO NOTHING
    `, payload.TransactionID, ledgerID, payload.ExternalID, "0", payload.Currency, occurredAt)
	if err != nil {
		return fmt.Errorf("insert transaction failed: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return nil
	}

	// Process postings
	for _, posting := range payload.Postings {
		accountCode := posting.AccountCode
		direction := posting.Direction
		amount := posting.Amount

		// TODO: Find AccountID, using cache if possible
		var accountID string
		err = tx.QueryRow(ctx, `
          SELECT id FROM accounts WHERE ledger_id = $1 AND code = $2
       `, ledgerID, accountCode).Scan(&accountID)

		if err != nil {
			return fmt.Errorf("account %s not found: %w", accountCode, err)
		}

		// Persist Posting Log
		postingID := uuid.NewString()
		_, err = tx.Exec(ctx, `
			INSERT INTO postings (
				id,
				ledger_id,
				transaction_id,
				account_id,
				amount,
				direction
			) VALUES ($1, $2, $3, $4, $5, $6)
		`, postingID, ledgerID, payload.TransactionID, accountID, amount, direction)
		if err != nil {
			return fmt.Errorf("insert posting failed: %w", err)
		}

		// Update account balance
		if err := p.updateAccountBalance(ctx, tx, accountID, direction, amount); err != nil {
			return err
		}
	}

	return nil
}

func (p *Projector) updateAccountBalance(ctx context.Context, tx pgx.Tx, accountID, direction, amountStr string) error {
	amount := new(big.Rat)
	if _, ok := amount.SetString(amountStr); !ok {
		return fmt.Errorf("invalid amount: %s", amountStr)
	}

	var finalAmount *big.Rat
	if direction == "credit" {
		finalAmount = amount
	} else {
		finalAmount = new(big.Rat).Neg(amount)
	}

	_, err := tx.Exec(ctx, `
       UPDATE accounts 
       SET balance = balance + $1 
       WHERE id = $2
    `, finalAmount.FloatString(10), accountID)

	return err
}

func (p *Projector) eventStore() events.EventBoundary {
	if p.EventStore != nil {
		return p.EventStore
	}
	return events.NewPGXStore()
}
