package ledger

import (
	"Go_FormanceLegder/internal/auth"
	"encoding/json"
	"net/http"
	"time"
)

type Handler struct {
	Service *Service
}

type PostTransactionRequest struct {
	IdempotencyKey string         `json:"idempotency_key"`
	ExternalID     string         `json:"external_id"`
	Currency       string         `json:"currency"`
	OccurredAt     time.Time      `json:"occurred_at"`
	Postings       []PostingInput `json:"postings"`
}

type PostTransactionResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

func (h *Handler) PostTransaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req PostTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	cmd := PostTransactionCommand{
		LedgerID:       principal.LedgerID,
		ExternalID:     req.ExternalID,
		IdempotencyKey: req.IdempotencyKey,
		Currency:       req.Currency,
		OccurredAt:     req.OccurredAt,
		Postings:       req.Postings,
	}

	transactionID, err := h.Service.PostTransaction(ctx, cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := PostTransactionResponse{
		TransactionID: transactionID,
		Status:        "accepted",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
