package dashboardauth

import (
	"Go_FormanceLegder/internal/auth"
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Action string

const (
	ActionRead  Action = "read"
	ActionWrite Action = "write"
)

type ResourceKind string

const (
	KindOrg     ResourceKind = "org"
	KindProject ResourceKind = "project"
	KindLedger  ResourceKind = "ledger"
	KindAPIKey  ResourceKind = "api_key"
)

type Target struct {
	Kind    ResourceKind
	IDParam string
	Action  Action
}

type Scope struct {
	UserID    string
	OrgID     string
	Role      string
	ProjectID string
	LedgerID  string
	APIKeyID  string
}

type Guard interface {
	Must(w http.ResponseWriter, r *http.Request, target Target) (Scope, bool)
	FromContext(ctx context.Context) (Scope, bool)
}

type contextKey string

const scopeKey contextKey = "dashboard_scope"

type PGXGuard struct {
	DB        *pgxpool.Pool
	JWTSecret []byte
}

func NewGuard(db *pgxpool.Pool, jwtSecret []byte) *PGXGuard {
	return &PGXGuard{
		DB:        db,
		JWTSecret: jwtSecret,
	}
}

func (g *PGXGuard) Must(w http.ResponseWriter, r *http.Request, target Target) (Scope, bool) {
	ctx := r.Context()

	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return Scope{}, false
	}

	claims, err := auth.ValidateJWT(cookie.Value, g.JWTSecret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return Scope{}, false
	}

	scope := Scope{
		UserID: claims.UserID,
		OrgID:  claims.OrgID,
	}

	// Load current role for policy checks or downstream use.
	_ = g.DB.QueryRow(ctx, `
		SELECT role
		FROM org_users
		WHERE user_id = $1
		  AND organization_id = $2
		LIMIT 1
	`, scope.UserID, scope.OrgID).Scan(&scope.Role)

	if target.Kind != KindOrg {
		id := ""
		if target.IDParam != "" {
			id = r.URL.Query().Get(target.IDParam)
			if id == "" {
				http.Error(w, fmt.Sprintf("%s required", target.IDParam), http.StatusBadRequest)
				return Scope{}, false
			}
		}

		ok, ownErr := g.belongsToOrg(ctx, scope.OrgID, target.Kind, id)
		if ownErr != nil || !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return Scope{}, false
		}

		switch target.Kind {
		case KindProject:
			scope.ProjectID = id
		case KindLedger:
			scope.LedgerID = id
		case KindAPIKey:
			scope.APIKeyID = id
		}
	}

	ctx = context.WithValue(ctx, scopeKey, scope)
	*r = *r.WithContext(ctx)
	return scope, true
}

func (g *PGXGuard) FromContext(ctx context.Context) (Scope, bool) {
	scope, ok := ctx.Value(scopeKey).(Scope)
	return scope, ok
}

func (g *PGXGuard) belongsToOrg(ctx context.Context, orgID string, kind ResourceKind, id string) (bool, error) {
	query := ""
	switch kind {
	case KindProject:
		query = `SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1 AND organization_id = $2)`
	case KindLedger:
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM ledgers l
				JOIN projects p ON p.id = l.project_id
				WHERE l.id = $1 AND p.organization_id = $2
			)
		`
	case KindAPIKey:
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM api_keys k
				JOIN ledgers l ON l.id = k.ledger_id
				JOIN projects p ON p.id = l.project_id
				WHERE k.id = $1 AND p.organization_id = $2
			)
		`
	default:
		return false, fmt.Errorf("unsupported resource kind: %s", kind)
	}

	var ok bool
	if err := g.DB.QueryRow(ctx, query, id, orgID).Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}
