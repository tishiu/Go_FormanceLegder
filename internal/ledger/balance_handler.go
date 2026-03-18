package ledger

import (
	"Go_FormanceLegder/internal/auth"
	"encoding/json"
	"net/http"
)

type BalanceSummaryResponse struct {
	TotalAssets      string            `json:"total_assets"`
	TotalLiabilities string            `json:"total_liabilities"`
	TotalEquity      string            `json:"total_equity"`
	TotalRevenue     string            `json:"total_revenue"`
	TotalExpenses    string            `json:"total_expenses"`
	ByType           map[string]string `json:"by_type"`
}

// GET /v1/balance/summary - Get balance summary by account type
func (h *Handler) GetBalanceSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	summary, err := h.scope(principal.LedgerID).GetBalanceSummary(ctx)
	if err != nil {
		http.Error(w, "failed to query balances", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

type AccountBalanceHistoryResponse struct {
	AccountCode string                `json:"account_code"`
	History     []BalanceHistoryPoint `json:"history"`
}

type BalanceHistoryPoint struct {
	Date    string `json:"date"`
	Balance string `json:"balance"`
}

// GET /v1/accounts/:code/balance-history - Get balance history for an account
func (h *Handler) GetAccountBalanceHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	accountCode := r.URL.Query().Get("code")
	if accountCode == "" {
		http.Error(w, "account code required", http.StatusBadRequest)
		return
	}

	response, err := h.scope(principal.LedgerID).GetAccountBalanceHistory(ctx, accountCode)
	if err != nil {
		http.Error(w, "account not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
