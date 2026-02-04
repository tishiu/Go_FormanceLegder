package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Principal struct {
	APIKeyID       string
	OrganizationID string
	ProjectID      string
	LedgerID       string
}

type contextKey string

const principalKey contextKey = "principal"

type Middleware struct {
	DB           *pgxpool.Pool
	APIKeySecret []byte
}

func (m *Middleware) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("Authorization")
		if raw == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(raw, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		apiKey := strings.TrimSpace(parts[1])
		if apiKey == "" {
			http.Error(w, "invalid api key", http.StatusUnauthorized)
			return
		}

		keyHash, err := ComputeKeyHash(m.APIKeySecret, apiKey)
		if err != nil {
			http.Error(w, "invalid api key", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		row := m.DB.QueryRow(ctx, `
			SELECT k.id, l.id, p.id, o.id
			FROM api_keys k
			JOIN ledgers l ON l.id = k.ledger_id
			JOIN projects p ON p.id = l.project_id
			JOIN organizations o ON o.id = p.organization_id
			WHERE k.key_hash = $1
			  AND k.is_active = true
			  AND k.revoked_at IS NULL
		`, keyHash)

		var principal Principal
		err = row.Scan(&principal.APIKeyID, &principal.LedgerID, &principal.ProjectID, &principal.OrganizationID)
		if err != nil {
			http.Error(w, "invalid api key", http.StatusUnauthorized)
			return
		}

		ctx = context.WithValue(ctx, principalKey, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func FromContext(ctx context.Context) (Principal, error) {
	p, ok := ctx.Value(principalKey).(Principal)
	if !ok {
		return Principal{}, errors.New("missing principal")
	}
	return p, nil
}

func ComputeKeyHash(secret []byte, key string) (string, error) {
	h := hmac.New(sha256.New, secret)
	_, err := h.Write([]byte(key))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}
