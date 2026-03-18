package dashboard

import (
	"Go_FormanceLegder/internal/auth"
	"Go_FormanceLegder/internal/dashboardauth"
	"encoding/base32"
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKeyHandler struct {
	DB           *pgxpool.Pool
	APIKeySecret []byte
	Guard        dashboardauth.Guard
}

type APIKeyResponse struct {
	ID          string `json:"id"`
	Prefix      string `json:"prefix"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	RevokedAt   string `json:"revoked_at,omitempty"`
}

type CreateAPIKeyRequest struct {
	Description string `json:"description"`
}

type CreateAPIKeyResponse struct {
	ID          string `json:"id"`
	RawKey      string `json:"raw_key"`
	Prefix      string `json:"prefix"`
	Description string `json:"description"`
}

// GET /api/ledgers/:ledgerId/api-keys
func (h *APIKeyHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.Guard == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	scope, ok := h.Guard.Must(w, r, dashboardauth.Target{
		Kind:    dashboardauth.KindLedger,
		IDParam: "ledger_id",
	})
	if !ok {
		return
	}

	rows, err := h.DB.Query(ctx, `
		SELECT id, prefix, description, is_active, created_at, revoked_at
		FROM api_keys
		WHERE ledger_id = $1
		ORDER BY created_at DESC
	`, scope.LedgerID)
	if err != nil {
		http.Error(w, "failed to query api keys", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	keys := []APIKeyResponse{}
	for rows.Next() {
		var key APIKeyResponse
		var revokedAt *string
		err = rows.Scan(&key.ID, &key.Prefix, &key.Description, &key.IsActive, &key.CreatedAt, &revokedAt)
		if err != nil {
			http.Error(w, "failed to scan api key", http.StatusInternalServerError)
			return
		}
		if revokedAt != nil {
			key.RevokedAt = *revokedAt
		}
		keys = append(keys, key)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

// POST /api/ledgers/:ledgerId/api-keys
func (h *APIKeyHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.Guard == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	scope, ok := h.Guard.Must(w, r, dashboardauth.Target{
		Kind:    dashboardauth.KindLedger,
		IDParam: "ledger_id",
	})
	if !ok {
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Generate raw API key
	rawKey, err := generateAPIKey()
	if err != nil {
		http.Error(w, "failed to generate api key", http.StatusInternalServerError)
		return
	}

	// Compute hash
	keyHash, err := auth.ComputeKeyHash(h.APIKeySecret, rawKey)
	if err != nil {
		http.Error(w, "failed to hash api key", http.StatusInternalServerError)
		return
	}

	// Extract prefix (first 10 characters)
	prefix := rawKey[:10]

	// Store in database
	var keyID string
	err = h.DB.QueryRow(ctx, `
		INSERT INTO api_keys (ledger_id, key_hash, prefix, description, is_active)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id
	`, scope.LedgerID, keyHash, prefix, req.Description).Scan(&keyID)
	if err != nil {
		http.Error(w, "failed to create api key", http.StatusInternalServerError)
		return
	}

	resp := CreateAPIKeyResponse{
		ID:          keyID,
		RawKey:      rawKey,
		Prefix:      prefix,
		Description: req.Description,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// POST /api/api-keys/:id/revoke
func (h *APIKeyHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.Guard == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	scope, ok := h.Guard.Must(w, r, dashboardauth.Target{
		Kind:    dashboardauth.KindAPIKey,
		IDParam: "id",
	})
	if !ok {
		return
	}

	// Revoke key
	_, err := h.DB.Exec(ctx, `
		UPDATE api_keys
		SET is_active = false, revoked_at = NOW()
		WHERE id = $1
	`, scope.APIKeyID)
	if err != nil {
		http.Error(w, "failed to revoke api key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func generateAPIKey() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode as base32 (URL-safe)
	encoded := base32.StdEncoding.EncodeToString(bytes)
	encoded = strings.TrimRight(encoded, "=") // Remove padding

	// Format: sk_live_<encoded>
	return "sk_live_" + strings.ToLower(encoded), nil
}
