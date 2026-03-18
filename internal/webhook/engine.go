package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeliverEventRequest struct {
	LedgerID string
	EventID  string
	Attempt  int
}

type DeliverEventResult struct {
	Delivered         int
	SkippedIdempotent int
	RetryableFailures int
	PermanentFailures int
}

type DeliveryEngine interface {
	DeliverEvent(ctx context.Context, req DeliverEventRequest) (DeliverEventResult, error)
}

type DefaultDeliveryEngine struct {
	DB         *pgxpool.Pool
	HttpClient *http.Client
}

func NewDefaultDeliveryEngine(db *pgxpool.Pool, httpClient *http.Client) *DefaultDeliveryEngine {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &DefaultDeliveryEngine{
		DB:         db,
		HttpClient: httpClient,
	}
}

func (e *DefaultDeliveryEngine) DeliverEvent(ctx context.Context, req DeliverEventRequest) (DeliverEventResult, error) {
	result := DeliverEventResult{}

	// Load event payload.
	var payloadJSON []byte
	err := e.DB.QueryRow(ctx, `
        SELECT payload
        FROM events
        WHERE id = $1 AND ledger_id = $2
    `, req.EventID, req.LedgerID).Scan(&payloadJSON)
	if err != nil {
		return result, fmt.Errorf("event not found (id=%s, ledger=%s): %w", req.EventID, req.LedgerID, err)
	}

	// Load active endpoints.
	rows, err := e.DB.Query(ctx, `
		SELECT id, url, secret
		FROM webhook_endpoints
		WHERE ledger_id = $1
		  AND is_active = true
	`, req.LedgerID)
	if err != nil {
		return result, fmt.Errorf("failed to load endpoints: %w", err)
	}
	defer rows.Close()

	endpoints := []WebhookEndpoint{}
	for rows.Next() {
		var ep WebhookEndpoint
		if err := rows.Scan(&ep.ID, &ep.URL, &ep.Secret); err == nil {
			endpoints = append(endpoints, ep)
		}
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	if len(endpoints) == 0 {
		return result, nil
	}

	for _, ep := range endpoints {
		var alreadySent bool
		err := e.DB.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM webhook_deliveries
				WHERE event_id = $1
				  AND webhook_endpoint_id = $2
				  AND status = 'success'
			)
		`, req.EventID, ep.ID).Scan(&alreadySent)
		if err != nil {
			result.RetryableFailures++
			continue
		}
		if alreadySent {
			result.SkippedIdempotent++
			continue
		}

		shouldRetry, sendErr := e.sendSingleWebhook(ctx, ep, req.EventID, payloadJSON, req.Attempt)
		if sendErr != nil {
			if shouldRetry {
				result.RetryableFailures++
			} else {
				result.PermanentFailures++
			}
			continue
		}
		result.Delivered++
	}

	if result.RetryableFailures > 0 {
		return result, fmt.Errorf("webhook delivery had %d retryable failures", result.RetryableFailures)
	}
	return result, nil
}

func (e *DefaultDeliveryEngine) sendSingleWebhook(ctx context.Context, ep WebhookEndpoint, eventID string, payload []byte, attempt int) (bool, error) {
	sig := computeWebhookSignature([]byte(ep.Secret), payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, bytes.NewReader(payload))
	if err != nil {
		e.logDelivery(ctx, eventID, ep.ID, "non_retryable_error", attempt, 0, err.Error())
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Ledger-Signature", sig)
	req.Header.Set("User-Agent", "LedgerKiro-Webhook/1.0")

	resp, err := e.HttpClient.Do(req)

	status := "success"
	httpStatus := 0
	errorMessage := ""
	shouldRetry := false

	if err != nil {
		status = "retryable_error"
		errorMessage = err.Error()
		shouldRetry = true
	} else {
		httpStatus = resp.StatusCode
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode >= 500 {
			status = "retryable_error"
			errorMessage = fmt.Sprintf("server error: %d", resp.StatusCode)
			shouldRetry = true
		} else if resp.StatusCode >= 400 {
			status = "non_retryable_error"
			errorMessage = fmt.Sprintf("client error: %d", resp.StatusCode)
			shouldRetry = false
		}
	}

	e.logDelivery(ctx, eventID, ep.ID, status, attempt, httpStatus, errorMessage)
	if shouldRetry {
		return true, fmt.Errorf("retryable failure for %s: %s", ep.URL, errorMessage)
	}
	return false, nil
}

func (e *DefaultDeliveryEngine) logDelivery(ctx context.Context, eventID, endpointID, status string, attempt, httpStatus int, errorMessage string) {
	_, _ = e.DB.Exec(ctx, `
		INSERT INTO webhook_deliveries (
			id,
			event_id,
			webhook_endpoint_id,
			status,
			attempt,
			last_attempt_at,
			http_status,
			error_message
		) VALUES ($1, $2, $3, $4, $5, NOW(), $6, $7)
	`, uuid.NewString(), eventID, endpointID, status, attempt, httpStatus, errorMessage)
}

func computeWebhookSignature(secret []byte, payload []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum)
}
