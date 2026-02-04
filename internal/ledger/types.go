package ledger

import "time"

type PostingInput struct {
	AccountCode string `json:"account_code"`
	Direction   string `json:"direction"`
	Amount      string `json:"amount"`
}

type PostTransactionCommand struct {
	LedgerID       string
	ExternalID     string
	IdempotencyKey string
	Currency       string
	Postings       []PostingInput
	OccurredAt     time.Time
}

type Account struct {
	ID      string
	Code    string
	Type    string
	Balance string
}
