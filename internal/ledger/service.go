package ledger

import (
	"Go_FormanceLegder/internal/webhook"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type Service struct {
	DB          *pgxpool.Pool
	RiverClient *river.Client[pgx.Tx]
}

func NewService(db *pgxpool.Pool, riverClient *river.Client[pgx.Tx]) *Service {
	return &Service{
		DB:          db,
		RiverClient: riverClient,
	}
}

func (s *Service) PostTransaction(ctx context.Context, cmd PostTransactionCommand) (string, error) {
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	// Check idempotency
	var existingID string
	err = tx.QueryRow(ctx, `
		SELECT aggregate_id
		FROM events
		WHERE ledger_id = $1
		  AND idempotency_key = $2
	`, cmd.LedgerID, cmd.IdempotencyKey).Scan(&existingID)
	if err == nil {
		// Already processed
		return existingID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	// Load and lock accounts
	accounts, err := s.loadAndLockAccounts(ctx, tx, cmd.LedgerID, cmd.Postings)
	if err != nil {
		return "", err
	}

	// Validate double-entry
	if err := validateDoubleEntry(cmd, accounts); err != nil {
		return "", err
	}

	// Append event
	eventID := uuid.NewString()
	transactionID := uuid.NewString()

	payload := map[string]any{
		"transaction_id": transactionID,
		"external_id":    cmd.ExternalID,
		"currency":       cmd.Currency,
		"occurred_at":    cmd.OccurredAt.UTC().Format(time.RFC3339Nano),
		"postings":       cmd.Postings,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO events (
			id,
			ledger_id,
			aggregate_type,
			aggregate_id,
			event_type,
			payload,
			occurred_at,
			idempotency_key
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, eventID, cmd.LedgerID, "ledger", transactionID, "TransactionPosted", payloadJSON, cmd.OccurredAt, cmd.IdempotencyKey)
	if err != nil {
		return "", err
	}

	// Enqueue webhook job atomically
	_, err = s.RiverClient.InsertTx(ctx, tx, webhook.WebhookArgs{
		EventID:  eventID,
		LedgerID: cmd.LedgerID,
	}, nil)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return transactionID, nil
}

func (s *Service) loadAndLockAccounts(ctx context.Context, tx pgx.Tx, ledgerID string, postings []PostingInput) (map[string]Account, error) {
	codesSet := map[string]struct{}{}
	for _, p := range postings {
		codesSet[p.AccountCode] = struct{}{}
	}
	codes := make([]string, 0, len(codesSet))
	for c := range codesSet {
		codes = append(codes, c)
	}
	sort.Strings(codes) // Deterministic lock order

	rows, err := tx.Query(ctx, `
		SELECT id, code, type, balance
		FROM accounts
		WHERE ledger_id = $1
		  AND code = ANY($2)
		FOR UPDATE
	`, ledgerID, codes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := map[string]Account{}
	for rows.Next() {
		var a Account
		err = rows.Scan(&a.ID, &a.Code, &a.Type, &a.Balance)
		if err != nil {
			return nil, err
		}
		accounts[a.Code] = a
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(accounts) != len(codes) {
		return nil, fmt.Errorf("one or more accounts not found for ledger %s", ledgerID)
	}

	return accounts, nil
}
