package ledger

import (
	"Go_FormanceLegder/internal/auth"
	"encoding/json"
	"net/http"
)

type AccountResponse struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Balance   string `json:"balance"`
	CreatedAt string `json:"created_at"`
}

// GET /v1/accounts - List all accounts for the authenticated ledger
func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := h.Service.DB.Query(ctx, `
		SELECT id, code, name, type, balance, created_at
		FROM accounts
		WHERE ledger_id = $1
		ORDER BY code
	`, principal.LedgerID)
	if err != nil {
		http.Error(w, "failed to query accounts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	accounts := []AccountResponse{}
	for rows.Next() {
		var acc AccountResponse
		err = rows.Scan(&acc.ID, &acc.Code, &acc.Name, &acc.Type, &acc.Balance, &acc.CreatedAt)
		if err != nil {
			http.Error(w, "failed to scan account", http.StatusInternalServerError)
			return
		}
		accounts = append(accounts, acc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

// GET /v1/accounts/:code - Get a specific account by code
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract account code from URL path or query param
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "account code required", http.StatusBadRequest)
		return
	}

	var acc AccountResponse
	err = h.Service.DB.QueryRow(ctx, `
		SELECT id, code, name, type, balance, created_at
		FROM accounts
		WHERE ledger_id = $1 AND code = $2
	`, principal.LedgerID, code).Scan(&acc.ID, &acc.Code, &acc.Name, &acc.Type, &acc.Balance, &acc.CreatedAt)
	if err != nil {
		http.Error(w, "account not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(acc)
}

// POST /v1/accounts - Create a new account
func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Code string `json:"code"`
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Validate account type
	validTypes := map[string]bool{
		"asset": true, "liability": true, "equity": true, "revenue": true, "expense": true,
	}
	if !validTypes[req.Type] {
		http.Error(w, "invalid account type", http.StatusBadRequest)
		return
	}

	var accountID string
	err = h.Service.DB.QueryRow(ctx, `
		INSERT INTO accounts (ledger_id, code, name, type, balance)
		VALUES ($1, $2, $3, $4, 0)
		RETURNING id
	`, principal.LedgerID, req.Code, req.Name, req.Type).Scan(&accountID)
	if err != nil {
		http.Error(w, "failed to create account", http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"id":   accountID,
		"code": req.Code,
		"name": req.Name,
		"type": req.Type,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
