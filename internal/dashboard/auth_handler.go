package dashboard

import (
	"Go_FormanceLegder/internal/auth"
	"Go_FormanceLegder/internal/config"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandler struct {
	DB     *pgxpool.Pool
	Config *config.Config
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID               string `json:"id"`
	Email            string `json:"email"`
	OrganizationID   string `json:"organization_id"`
	Role             string `json:"role"`
	DefaultProjectID string `json:"default_project_id,omitempty"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// POST /api/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 6 {
		http.Error(w, "password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "failed to process password", http.StatusInternalServerError)
		return
	}

	tx, err := h.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		http.Error(w, "failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id
	`, req.Email, passwordHash).Scan(&userID)
	if err != nil {
		if isUniqueViolation(err) {
			http.Error(w, "email already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	var orgID string
	orgName := req.Email + "'s Organization"
	err = tx.QueryRow(ctx, `
		INSERT INTO organizations (name)
		VALUES ($1)
		RETURNING id
	`, orgName).Scan(&orgID)
	if err != nil {
		http.Error(w, "failed to create organization", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO org_users (organization_id, user_id, role)
		VALUES ($1, $2, 'owner')
	`, orgID, userID)
	if err != nil {
		http.Error(w, "failed to link organization", http.StatusInternalServerError)
		return
	}

	var defaultProjectID string
	err = tx.QueryRow(ctx, `
		INSERT INTO projects (organization_id, name, code)
		VALUES ($1, 'Default Project', 'default')
		RETURNING id
	`, orgID).Scan(&defaultProjectID)
	if err != nil {
		http.Error(w, "failed to create default project", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "failed to commit registration", http.StatusInternalServerError)
		return
	}

	token, err := auth.GenerateJWT(userID, orgID, h.Config.SessionTimeout, h.Config.JWTSecret)
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, token, h.Config.SessionTimeout)

	resp := AuthResponse{
		Token: token,
		User: UserResponse{
			ID:               userID,
			Email:            req.Email,
			OrganizationID:   orgID,
			Role:             "owner",
			DefaultProjectID: defaultProjectID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}

	var (
		userID           string
		passwordHash     string
		orgID            string
		role             string
		defaultProjectID string
	)
	err := h.DB.QueryRow(ctx, `
		SELECT 
			u.id,
			u.password_hash,
			ou.organization_id,
			ou.role,
			COALESCE(p.id, '')
		FROM users u
		JOIN org_users ou ON ou.user_id = u.id
		LEFT JOIN projects p ON p.organization_id = ou.organization_id AND p.code = 'default'
		WHERE u.email = $1
		ORDER BY ou.created_at
		LIMIT 1
	`, req.Email).Scan(&userID, &passwordHash, &orgID, &role, &defaultProjectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}

	if err := auth.CheckPassword(passwordHash, req.Password); err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateJWT(userID, orgID, h.Config.SessionTimeout, h.Config.JWTSecret)
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, token, h.Config.SessionTimeout)

	resp := AuthResponse{
		Token: token,
		User: UserResponse{
			ID:               userID,
			Email:            req.Email,
			OrganizationID:   orgID,
			Role:             role,
			DefaultProjectID: defaultProjectID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GET /api/me
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract JWT from cookie
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	claims, err := auth.ValidateJWT(cookie.Value, h.Config.JWTSecret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var user UserResponse
	// Fetch default project ID as well
	err = h.DB.QueryRow(ctx, `
		SELECT u.id, u.email, ou.organization_id, ou.role, COALESCE(p.id, '')
		FROM users u
		JOIN org_users ou ON ou.user_id = u.id
        LEFT JOIN projects p ON p.organization_id = ou.organization_id AND p.code = 'default'
		WHERE u.id = $1 AND ou.organization_id = $2
	`, claims.UserID, claims.OrgID).Scan(&user.ID, &user.Email, &user.OrganizationID, &user.Role, &user.DefaultProjectID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func setSessionCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
}
