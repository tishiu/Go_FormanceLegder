package dashboard

import (
	"Go_FormanceLegder/internal/auth"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WebhookHandler struct {
	DB *pgxpool.Pool
}

type WebhookEndpointResponse struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

type CreateWebhookEndpointRequest struct {
	URL string `json:"url"`
}

type CreateWebhookEndpointResponse struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

type WebhookDeliveryResponse struct {
	ID                string `json:"id"`
	EventID           string `json:"event_id"`
	WebhookEndpointID string `json:"webhook_endpoint_id"`
	EndpointURL       string `json:"endpoint_url"`
	Status            string `json:"status"`
	Attempt           int    `json:"attempt"`
	LastAttemptAt     string `json:"last_attempt_at"`
	HTTPStatus        int    `json:"http_status"`
	ErrorMessage      string `json:"error_message,omitempty"`
}

// GET /v1/webhook-endpoints
func (h *WebhookHandler) ListWebhookEndpoints(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := h.DB.Query(ctx, `
		SELECT id, url, is_active, created_at
		FROM webhook_endpoints
		WHERE ledger_id = $1
		ORDER BY created_at DESC
	`, principal.LedgerID)
	if err != nil {
		http.Error(w, "failed to query webhook endpoints", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	endpoints := []WebhookEndpointResponse{}
	for rows.Next() {
		var endpoint WebhookEndpointResponse
		err = rows.Scan(&endpoint.ID, &endpoint.URL, &endpoint.IsActive, &endpoint.CreatedAt)
		if err != nil {
			http.Error(w, "failed to scan webhook endpoint", http.StatusInternalServerError)
			return
		}
		endpoints = append(endpoints, endpoint)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endpoints)
}

// POST /v1/webhook-endpoints
func (h *WebhookHandler) CreateWebhookEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateWebhookEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Generate webhook secret
	secret, err := generateWebhookSecret()
	if err != nil {
		http.Error(w, "failed to generate secret", http.StatusInternalServerError)
		return
	}

	// Create endpoint
	var endpointID string
	err = h.DB.QueryRow(ctx, `
		INSERT INTO webhook_endpoints (ledger_id, url, secret, is_active)
		VALUES ($1, $2, $3, true)
		RETURNING id
	`, principal.LedgerID, req.URL, secret).Scan(&endpointID)
	if err != nil {
		http.Error(w, "failed to create webhook endpoint", http.StatusInternalServerError)
		return
	}

	resp := CreateWebhookEndpointResponse{
		ID:     endpointID,
		URL:    req.URL,
		Secret: secret,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// GET /v1/webhook-deliveries
func (h *WebhookHandler) ListWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse limit
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	if limit > 1000 {
		limit = 1000
	}

	rows, err := h.DB.Query(ctx, `
		SELECT 
			wd.id, 
			wd.event_id, 
			wd.webhook_endpoint_id, 
			we.url,
			wd.status, 
			wd.attempt, 
			wd.last_attempt_at, 
			wd.http_status, 
			wd.error_message
		FROM webhook_deliveries wd
		JOIN webhook_endpoints we ON we.id = wd.webhook_endpoint_id
		WHERE we.ledger_id = $1
		ORDER BY wd.last_attempt_at DESC
		LIMIT $2
	`, principal.LedgerID, limit)
	if err != nil {
		http.Error(w, "failed to query webhook deliveries", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	deliveries := []WebhookDeliveryResponse{}
	for rows.Next() {
		var delivery WebhookDeliveryResponse
		var errorMessage *string
		err = rows.Scan(
			&delivery.ID,
			&delivery.EventID,
			&delivery.WebhookEndpointID,
			&delivery.EndpointURL,
			&delivery.Status,
			&delivery.Attempt,
			&delivery.LastAttemptAt,
			&delivery.HTTPStatus,
			&errorMessage,
		)
		if err != nil {
			http.Error(w, "failed to scan webhook delivery", http.StatusInternalServerError)
			return
		}
		if errorMessage != nil {
			delivery.ErrorMessage = *errorMessage
		}
		deliveries = append(deliveries, delivery)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deliveries)
}

func generateWebhookSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "whsec_" + hex.EncodeToString(bytes), nil
}
