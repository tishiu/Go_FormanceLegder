package dashboard

import (
	"Go_FormanceLegder/internal/auth"
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

	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	claims, err := auth.ValidateJWT(cookie.Value, []byte("jwt-secret"))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ledgerID := r.URL.Query().Get("ledger_id")
	if ledgerID == "" {
		http.Error(w, "ledger_id required", http.StatusBadRequest)
		return
	}

	// Verify ledger belongs to user's organization
	var projectOrgID string
	err = h.DB.QueryRow(ctx, `
		SELECT p.organization_id
		FROM ledgers l
		JOIN projects p ON p.id = l.project_id
		WHERE l.id = $1
	`, ledgerID).Scan(&projectOrgID)
	if err != nil || projectOrgID != claims.OrgID {
		http.Error(w, "ledger not found", http.StatusNotFound)
		return
	}

	rows, err := h.DB.Query(ctx, `
		SELECT id, prefix, description, is_active, created_at, revoked_at
		FROM api_keys
		WHERE ledger_id = $1
		ORDER BY created_at DESC
	`, ledgerID)
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

	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	claims, err := auth.ValidateJWT(cookie.Value, []byte("jwt-secret"))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ledgerID := r.URL.Query().Get("ledger_id")
	if ledgerID == "" {
		http.Error(w, "ledger_id required", http.StatusBadRequest)
		return
	}

	// Verify ledger belongs to user's organization
	var projectOrgID string
	err = h.DB.QueryRow(ctx, `
		SELECT p.organization_id
		FROM ledgers l
		JOIN projects p ON p.id = l.project_id
		WHERE l.id = $1
	`, ledgerID).Scan(&projectOrgID)
	if err != nil || projectOrgID != claims.OrgID {
		http.Error(w, "ledger not found", http.StatusNotFound)
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
	`, ledgerID, keyHash, prefix, req.Description).Scan(&keyID)
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

	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	claims, err := auth.ValidateJWT(cookie.Value, []byte("jwt-secret"))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	keyID := r.URL.Query().Get("id")
	if keyID == "" {
		http.Error(w, "key id required", http.StatusBadRequest)
		return
	}

	// Verify key belongs to user's organization
	var projectOrgID string
	err = h.DB.QueryRow(ctx, `
		SELECT p.organization_id
		FROM api_keys k
		JOIN ledgers l ON l.id = k.ledger_id
		JOIN projects p ON p.id = l.project_id
		WHERE k.id = $1
	`, keyID).Scan(&projectOrgID)
	if err != nil || projectOrgID != claims.OrgID {
		http.Error(w, "api key not found", http.StatusNotFound)
		return
	}

	// Revoke key
	_, err = h.DB.Exec(ctx, `
		UPDATE api_keys
		SET is_active = false, revoked_at = NOW()
		WHERE id = $1
	`, keyID)
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
