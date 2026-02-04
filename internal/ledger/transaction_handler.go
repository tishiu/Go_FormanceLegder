package ledger

import (
	"Go_FormanceLegder/internal/api"
	"Go_FormanceLegder/internal/auth"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TransactionResponse struct {
	ID         string          `json:"id"`
	ExternalID string          `json:"external_id"`
	Amount     string          `json:"amount"`
	Currency   string          `json:"currency"`
	OccurredAt string          `json:"occurred_at"`
	CreatedAt  string          `json:"created_at"`
	Postings   []PostingDetail `json:"postings"`
}

type PostingDetail struct {
	ID          string `json:"id"`
	AccountCode string `json:"account_code"`
	AccountName string `json:"account_name"`
	Direction   string `json:"direction"`
	Amount      string `json:"amount"`
}

type ListTransactionsResponse struct {
	Transactions []TransactionResponse  `json:"transactions"`
	Pagination   api.PaginationResponse `json:"pagination"`
}

// GET /v1/transactions - List transactions with pagination
func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	limit = api.ValidateLimit(limit)

	continuationToken := r.URL.Query().Get("continuation_token")
	cursor, err := api.DecodeCursor(continuationToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse time range filters (optional)
	startTime := r.URL.Query().Get("start_time")
	endTime := r.URL.Query().Get("end_time")

	// Build query
	query := `
		SELECT t.id, t.external_id, t.amount, t.currency, t.occurred_at, t.created_at
		FROM transactions t
		WHERE t.ledger_id = $1
	`
	args := []interface{}{principal.LedgerID}
	argCount := 1

	// Add cursor condition
	if cursor.Timestamp.IsZero() == false {
		argCount++
		query += ` AND (t.created_at, t.id) < ($` + fmt.Sprintf("%d", argCount) + `, $` + fmt.Sprintf("%d", argCount+1) + `)`
		args = append(args, cursor.Timestamp, cursor.ID)
		argCount++
	}

	// Add time range filters
	if startTime != "" {
		argCount++
		query += ` AND t.occurred_at >= $` + fmt.Sprintf("%d", argCount)
		args = append(args, startTime)
	}
	if endTime != "" {
		argCount++
		query += ` AND t.occurred_at <= $` + fmt.Sprintf("%d", argCount)
		args = append(args, endTime)
	}

	// Order and limit (fetch limit + 1 to check if there are more)
	query += ` ORDER BY t.created_at DESC, t.id DESC LIMIT $` + fmt.Sprintf("%d", argCount+1)
	args = append(args, limit+1)

	rows, err := h.Service.DB.Query(ctx, query, args...)
	if err != nil {
		http.Error(w, "failed to query transactions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	transactions := []TransactionResponse{}
	var lastCreatedAt time.Time
	var lastID string

	for rows.Next() {
		var txn TransactionResponse
		var createdAt time.Time
		err = rows.Scan(&txn.ID, &txn.ExternalID, &txn.Amount, &txn.Currency, &txn.OccurredAt, &createdAt)
		if err != nil {
			http.Error(w, "failed to scan transaction", http.StatusInternalServerError)
			return
		}
		txn.CreatedAt = createdAt.Format(time.RFC3339)

		// Stop if we've reached the limit
		if len(transactions) >= limit {
			break
		}

		transactions = append(transactions, txn)
		lastCreatedAt = createdAt
		lastID = txn.ID
	}

	// Check if there are more results
	hasMore := false
	if err = rows.Err(); err == nil {
		if rows.Next() {
			hasMore = true
		}
	}

	// Generate continuation token
	var nextToken string
	if hasMore && len(transactions) > 0 {
		nextCursor := api.Cursor{
			Timestamp: lastCreatedAt,
			ID:        lastID,
		}
		nextToken, _ = api.EncodeCursor(nextCursor)
	}

	// Load postings for each transaction
	for i := range transactions {
		postings, err := h.loadPostings(ctx, principal.LedgerID, transactions[i].ID)
		if err != nil {
			http.Error(w, "failed to load postings", http.StatusInternalServerError)
			return
		}
		transactions[i].Postings = postings
	}

	response := ListTransactionsResponse{
		Transactions: transactions,
		Pagination: api.PaginationResponse{
			HasMore:           hasMore,
			ContinuationToken: nextToken,
			Count:             len(transactions),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /v1/transactions/:id - Get a specific transaction
func (h *Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	transactionID := r.URL.Query().Get("id")
	if transactionID == "" {
		http.Error(w, "transaction id required", http.StatusBadRequest)
		return
	}

	var txn TransactionResponse
	var createdAt time.Time
	err = h.Service.DB.QueryRow(ctx, `
		SELECT id, external_id, amount, currency, occurred_at, created_at
		FROM transactions
		WHERE ledger_id = $1 AND id = $2
	`, principal.LedgerID, transactionID).Scan(&txn.ID, &txn.ExternalID, &txn.Amount, &txn.Currency, &txn.OccurredAt, &createdAt)
	if err != nil {
		http.Error(w, "transaction not found", http.StatusNotFound)
		return
	}
	txn.CreatedAt = createdAt.Format(time.RFC3339)

	// Load postings
	postings, err := h.loadPostings(ctx, principal.LedgerID, txn.ID)
	if err != nil {
		http.Error(w, "failed to load postings", http.StatusInternalServerError)
		return
	}
	txn.Postings = postings

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txn)
}

func (h *Handler) loadPostings(ctx context.Context, ledgerID, transactionID string) ([]PostingDetail, error) {
	rows, err := h.Service.DB.Query(ctx, `
		SELECT p.id, a.code, a.name, p.direction, p.amount
		FROM postings p
		JOIN accounts a ON a.id = p.account_id
		WHERE p.ledger_id = $1 AND p.transaction_id = $2
		ORDER BY p.created_at
	`, ledgerID, transactionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	postings := []PostingDetail{}
	for rows.Next() {
		var p PostingDetail
		err = rows.Scan(&p.ID, &p.AccountCode, &p.AccountName, &p.Direction, &p.Amount)
		if err != nil {
			return nil, err
		}
		postings = append(postings, p)
	}

	return postings, rows.Err()
}
