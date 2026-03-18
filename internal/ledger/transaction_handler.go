package ledger

import (
	"Go_FormanceLegder/internal/api"
	"Go_FormanceLegder/internal/auth"
	"encoding/json"
	"fmt"
	"net/http"
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

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	transactions, pagination, err := h.scope(principal.LedgerID).ListTransactions(ctx, TransactionListParams{
		Limit:             limit,
		ContinuationToken: r.URL.Query().Get("continuation_token"),
		StartTime:         r.URL.Query().Get("start_time"),
		EndTime:           r.URL.Query().Get("end_time"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := ListTransactionsResponse{
		Transactions: transactions,
		Pagination:   pagination,
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

	txn, err := h.scope(principal.LedgerID).GetTransaction(ctx, transactionID)
	if err != nil {
		http.Error(w, "transaction not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txn)
}
