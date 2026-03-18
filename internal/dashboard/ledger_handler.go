package dashboard

import (
	"Go_FormanceLegder/internal/dashboardauth"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LedgerHandler struct {
	DB    *pgxpool.Pool
	Guard dashboardauth.Guard
}

type LedgerResponse struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Currency  string `json:"currency"`
	CreatedAt string `json:"created_at"`
}

type CreateLedgerRequest struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Currency  string `json:"currency"`
}

// GET /api/ledgers - List all ledgers for the authenticated user's organization
func (h *LedgerHandler) ListLedgers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.Guard == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	scope, ok := h.Guard.Must(w, r, dashboardauth.Target{
		Kind: dashboardauth.KindOrg,
	})
	if !ok {
		return
	}

	rows, err := h.DB.Query(ctx, `
		SELECT l.id, l.project_id, l.name, l.code, l.currency, l.created_at
		FROM ledgers l
		JOIN projects p ON p.id = l.project_id
		WHERE p.organization_id = $1
		ORDER BY l.created_at DESC
	`, scope.OrgID)
	if err != nil {
		http.Error(w, "failed to query ledgers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	ledgers := []LedgerResponse{}
	for rows.Next() {
		var ledger LedgerResponse
		err = rows.Scan(&ledger.ID, &ledger.ProjectID, &ledger.Name, &ledger.Code, &ledger.Currency, &ledger.CreatedAt)
		if err != nil {
			http.Error(w, "failed to scan ledger", http.StatusInternalServerError)
			return
		}
		ledgers = append(ledgers, ledger)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ledgers)
}

// GET /api/ledgers/:id - Get a specific ledger
func (h *LedgerHandler) GetLedger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.Guard == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	scope, ok := h.Guard.Must(w, r, dashboardauth.Target{
		Kind:    dashboardauth.KindLedger,
		IDParam: "id",
	})
	if !ok {
		return
	}

	var ledger LedgerResponse
	err := h.DB.QueryRow(ctx, `
		SELECT l.id, l.project_id, l.name, l.code, l.currency, l.created_at
		FROM ledgers l
		WHERE l.id = $1
	`, scope.LedgerID).Scan(&ledger.ID, &ledger.ProjectID, &ledger.Name, &ledger.Code, &ledger.Currency, &ledger.CreatedAt)
	if err != nil {
		http.Error(w, "ledger not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ledger)
}

// POST /api/ledgers - Create a new ledger
func (h *LedgerHandler) CreateLedger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.Guard == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	scope, ok := h.Guard.Must(w, r, dashboardauth.Target{
		Kind: dashboardauth.KindOrg,
	})
	if !ok {
		return
	}

	var req CreateLedgerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Verify project belongs to user's organization
	var projectOrgID string
	err := h.DB.QueryRow(ctx, `
		SELECT organization_id FROM projects WHERE id = $1
	`, req.ProjectID).Scan(&projectOrgID)
	if err != nil || projectOrgID != scope.OrgID {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	// Create ledger
	var ledgerID string
	err = h.DB.QueryRow(ctx, `
		INSERT INTO ledgers (project_id, name, code, currency)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, req.ProjectID, req.Name, req.Code, req.Currency).Scan(&ledgerID)
	if err != nil {
		http.Error(w, "failed to create ledger", http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"id":         ledgerID,
		"project_id": req.ProjectID,
		"name":       req.Name,
		"code":       req.Code,
		"currency":   req.Currency,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
