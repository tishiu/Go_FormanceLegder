package projector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Projector struct {
	DB *pgxpool.Pool
}

func NewProjector(db *pgxpool.Pool) *Projector {
	return &Projector{DB: db}
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

	// Load Events
	type EventData struct {
		ID, LedgerID, Type string
		Payload            []byte
	}
	var events []EventData

	rows, err := tx.Query(ctx, `
       SELECT id, ledger_id, event_type, payload
       FROM events
       WHERE event_type = 'TransactionPosted'
         AND id > COALESCE((SELECT last_processed_event_id FROM projector_offsets WHERE projector_name = 'ledger'), '00000000-0000-0000-0000-000000000000')
       ORDER BY created_at, id
       LIMIT 100
    `)
	if err != nil {
		return err
	}
	for rows.Next() {
		var e EventData
		if err := rows.Scan(&e.ID, &e.LedgerID, &e.Type, &e.Payload); err != nil {
			rows.Close() // Nhớ close nếu return sớm
			return err
		}
		events = append(events, e)
	}
	rows.Close()

	if len(events) == 0 {
		return tx.Commit(ctx)
	}

	// Process
	var maxEventID string
	for _, event := range events {
		var payload map[string]any
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("bad payload event %s: %w", event.ID, err)
		}

		// Pass tx xuống để xử lý
		if err := p.applyTransactionPosted(ctx, tx, event.LedgerID, payload); err != nil {
			return fmt.Errorf("failed apply event %s: %w", event.ID, err)
		}
		maxEventID = event.ID
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

func (p *Projector) applyTransactionPosted(ctx context.Context, tx pgx.Tx, ledgerID string, payload map[string]any) error {
	transactionID := payload["transaction_id"].(string)
	externalID, _ := payload["external_id"].(string)
	currency := payload["currency"].(string)
	occurredAtStr := payload["occurred_at"].(string)
	occurredAt, err := time.Parse(time.RFC3339Nano, occurredAtStr)
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
    `, transactionID, ledgerID, externalID, "0", currency, occurredAt)
	if err != nil {
		return fmt.Errorf("insert transaction failed: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return nil
	}

	// Process postings
	postings, ok := payload["postings"].([]any)
	if !ok {
		return fmt.Errorf("invalid postings payload")
	}

	for _, raw := range postings {
		pMap := raw.(map[string]any)
		accountCode := pMap["account_code"].(string)
		direction := pMap["direction"].(string)
		amount := pMap["amount"].(string)

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
		`, postingID, ledgerID, transactionID, accountID, amount, direction)
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
