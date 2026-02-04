package ledger

import (
	"Go_FormanceLegder/internal/auth"
	"encoding/json"
	"fmt"
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

	rows, err := h.Service.DB.Query(ctx, `
		SELECT type, SUM(balance) as total
		FROM accounts
		WHERE ledger_id = $1
		GROUP BY type
	`, principal.LedgerID)
	if err != nil {
		http.Error(w, "failed to query balances", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	summary := BalanceSummaryResponse{
		TotalAssets:      "0",
		TotalLiabilities: "0",
		TotalEquity:      "0",
		TotalRevenue:     "0",
		TotalExpenses:    "0",
		ByType:           make(map[string]string),
	}

	for rows.Next() {
		var accountType, total string
		err = rows.Scan(&accountType, &total)
		if err != nil {
			http.Error(w, "failed to scan balance", http.StatusInternalServerError)
			return
		}

		summary.ByType[accountType] = total

		switch accountType {
		case "asset":
			summary.TotalAssets = total
		case "liability":
			summary.TotalLiabilities = total
		case "equity":
			summary.TotalEquity = total
		case "revenue":
			summary.TotalRevenue = total
		case "expense":
			summary.TotalExpenses = total
		}
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

	// Get account ID
	var accountID string
	err = h.Service.DB.QueryRow(ctx, `
		SELECT id FROM accounts WHERE ledger_id = $1 AND code = $2
	`, principal.LedgerID, accountCode).Scan(&accountID)
	if err != nil {
		http.Error(w, "account not found", http.StatusNotFound)
		return
	}

	// Query posting history grouped by date
	rows, err := h.Service.DB.Query(ctx, `
		SELECT 
			DATE(t.occurred_at) as date,
			SUM(CASE WHEN p.direction = 'debit' THEN p.amount ELSE -p.amount END) as net_change
		FROM postings p
		JOIN transactions t ON t.id = p.transaction_id
		WHERE p.account_id = $1
		GROUP BY DATE(t.occurred_at)
		ORDER BY date ASC
	`, accountID)
	if err != nil {
		http.Error(w, "failed to query balance history", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	history := []BalanceHistoryPoint{}
	runningBalance := 0.0

	for rows.Next() {
		var date string
		var netChange float64
		err = rows.Scan(&date, &netChange)
		if err != nil {
			http.Error(w, "failed to scan history", http.StatusInternalServerError)
			return
		}

		runningBalance += netChange
		history = append(history, BalanceHistoryPoint{
			Date:    date,
			Balance: fmt.Sprintf("%.2f", runningBalance),
		})
	}

	response := AccountBalanceHistoryResponse{
		AccountCode: accountCode,
		History:     history,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
