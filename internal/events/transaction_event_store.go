package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Posting struct {
	AccountCode string `json:"account_code"`
	Direction   string `json:"direction"`
	Amount      string `json:"amount"`
}

type TransactionPosted struct {
	TransactionID         string    `json:"transaction_id"`
	ExternalID            string    `json:"external_id"`
	Currency              string    `json:"currency"`
	OccurredAtRFC3339Nano string    `json:"occurred_at"`
	Postings              []Posting `json:"postings"`
}

type TransactionPostedEvent struct {
	EventID  string
	LedgerID string
	Data     TransactionPosted
}

type AppendInput struct {
	LedgerID       string
	TransactionID  string
	ExternalID     string
	IdempotencyKey string
	Currency       string
	OccurredAt     time.Time
	Postings       []Posting
}

type EventBoundary interface {
	AppendTransactionPosted(ctx context.Context, tx pgx.Tx, in AppendInput) (TransactionPostedEvent, error)
	LoadTransactionPosted(ctx context.Context, tx pgx.Tx, afterEventID string, limit int) ([]TransactionPostedEvent, string, error)
}

type PGXStore struct{}

func NewPGXStore() *PGXStore {
	return &PGXStore{}
}

func (s *PGXStore) AppendTransactionPosted(ctx context.Context, tx pgx.Tx, in AppendInput) (TransactionPostedEvent, error) {
	eventID := uuid.NewString()
	payload := TransactionPosted{
		TransactionID:         in.TransactionID,
		ExternalID:            in.ExternalID,
		Currency:              in.Currency,
		OccurredAtRFC3339Nano: in.OccurredAt.UTC().Format(time.RFC3339Nano),
		Postings:              in.Postings,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return TransactionPostedEvent{}, err
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
	`, eventID, in.LedgerID, "ledger", in.TransactionID, "TransactionPosted", payloadJSON, in.OccurredAt, in.IdempotencyKey)
	if err != nil {
		return TransactionPostedEvent{}, err
	}

	return TransactionPostedEvent{
		EventID:  eventID,
		LedgerID: in.LedgerID,
		Data:     payload,
	}, nil
}

func (s *PGXStore) LoadTransactionPosted(ctx context.Context, tx pgx.Tx, afterEventID string, limit int) ([]TransactionPostedEvent, string, error) {
	if limit <= 0 {
		limit = 100
	}
	if afterEventID == "" {
		afterEventID = "00000000-0000-0000-0000-000000000000"
	}

	rows, err := tx.Query(ctx, `
		SELECT id, ledger_id, payload
		FROM events
		WHERE event_type = 'TransactionPosted'
		  AND id > $1
		ORDER BY created_at, id
		LIMIT $2
	`, afterEventID, limit)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	events := []TransactionPostedEvent{}
	lastEventID := ""

	for rows.Next() {
		var (
			eventID  string
			ledgerID string
			payload  []byte
			data     TransactionPosted
		)
		if err := rows.Scan(&eventID, &ledgerID, &payload); err != nil {
			return nil, "", err
		}
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, "", err
		}

		events = append(events, TransactionPostedEvent{
			EventID:  eventID,
			LedgerID: ledgerID,
			Data:     data,
		})
		lastEventID = eventID
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	return events, lastEventID, nil
}
