package dashboard

import (
	"Go_FormanceLegder/internal/auth"
	"Go_FormanceLegder/internal/config"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
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
	ID             string `json:"id"`
	Email          string `json:"email"`
	OrganizationID string `json:"organization_id"`
	Role           string `json:"role"`
}

// POST /api/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	// Begin transaction
	tx, err := h.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		http.Error(w, "failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	// Create user
	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id
	`, req.Email, passwordHash).Scan(&userID)
	if err != nil {
		http.Error(w, "email already exists", http.StatusConflict)
		return
	}

	// Create organization (auto-generate name from email)
	var orgID string
	orgName := req.Email
	if atIndex := strings.Index(req.Email, "@"); atIndex > 0 {
		orgName = req.Email[:atIndex] + "'s Organization"
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO organizations (name)
		VALUES ($1)
		RETURNING id
	`, orgName).Scan(&orgID)
	if err != nil {
		http.Error(w, "failed to create organization", http.StatusInternalServerError)
		return
	}

	// Link user to organization
	_, err = tx.Exec(ctx, `
		INSERT INTO org_users (organization_id, user_id, role)
		VALUES ($1, $2, 'owner')
	`, orgID, userID)
	if err != nil {
		http.Error(w, "failed to link user to organization", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Generate JWT
	token, err := auth.GenerateJWT(userID, orgID, h.Config.SessionTimeout, h.Config.JWTSecret)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.Config.SessionTimeout.Seconds()),
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"user_id":         userID,
		"organization_id": orgID,
	})
}

// POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var userID, passwordHash, orgID string
	err := h.DB.QueryRow(ctx, `
		SELECT u.id, u.password_hash, o.id
		FROM users u
		JOIN org_users ou ON ou.user_id = u.id
		JOIN organizations o ON o.id = ou.organization_id
		WHERE u.email = $1
		LIMIT 1
	`, req.Email).Scan(&userID, &passwordHash, &orgID)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := auth.CheckPassword(passwordHash, req.Password); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT
	token, err := auth.GenerateJWT(userID, orgID, h.Config.SessionTimeout, h.Config.JWTSecret)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.Config.SessionTimeout.Seconds()),
	})

	w.WriteHeader(http.StatusNoContent)
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
	err = h.DB.QueryRow(ctx, `
		SELECT u.id, u.email, ou.organization_id, ou.role
		FROM users u
		JOIN org_users ou ON ou.user_id = u.id
		WHERE u.id = $1 AND ou.organization_id = $2
	`, claims.UserID, claims.OrgID).Scan(&user.ID, &user.Email, &user.OrganizationID, &user.Role)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
