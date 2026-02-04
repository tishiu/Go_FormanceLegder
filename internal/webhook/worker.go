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
	"github.com/riverqueue/river"
)

type Worker struct {
	river.WorkerDefaults[WebhookArgs]
	DB         *pgxpool.Pool
	HttpClient *http.Client
}

func NewWorker(db *pgxpool.Pool) *Worker {
	return &Worker{
		DB: db,
		HttpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (w *Worker) Work(ctx context.Context, job *river.Job[WebhookArgs]) error {
	args := job.Args

	// Load event payload
	var payloadJSON []byte
	err := w.DB.QueryRow(ctx, `
        SELECT payload
        FROM events
        WHERE id = $1 AND ledger_id = $2
    `, args.EventID, args.LedgerID).Scan(&payloadJSON)

	if err != nil {
		return fmt.Errorf("event not found (id=%s, ledger=%s): %w", args.EventID, args.LedgerID, err)
	}

	// Load active webhook endpoints
	rows, err := w.DB.Query(ctx, `
		SELECT id, url, secret
		FROM webhook_endpoints
		WHERE ledger_id = $1
		  AND is_active = true
	`, args.LedgerID)
	if err != nil {
		return fmt.Errorf("failed to load endpoints: %w", err)
	}

	var endpoints []WebhookEndpoint
	for rows.Next() {
		var ep WebhookEndpoint
		if err := rows.Scan(&ep.ID, &ep.URL, &ep.Secret); err == nil {
			endpoints = append(endpoints, ep)
		}
	}
	defer rows.Close()

	if len(endpoints) == 0 {
		return nil
	}

	// Deliver to each endpoint with idempotency checks.
	var retryableFailures int

	for _, ep := range endpoints {
		// Idempotency: if already delivered successfully for this (event, endpoint), skip.
		var alreadySent bool
		err := w.DB.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM webhook_deliveries
				WHERE event_id = $1
				  AND webhook_endpoint_id = $2
				  AND status = 'success'
			)
		`, args.EventID, ep.ID).Scan(&alreadySent)
		if err != nil {
			// Treat DB check errors as retryable: job should retry.
			retryableFailures++
			continue
		}
		if alreadySent {
			continue
		}

		// Send single webhook and record delivery result.
		shouldRetry, sendErr := w.sendSingleWebhook(ctx, ep, args.EventID, payloadJSON, job.Attempt)
		if sendErr != nil {
			// sendErr is informational here; delivery was logged. We decide retry based on shouldRetry.
			if shouldRetry {
				retryableFailures++
			}
		}
	}

	// 4) Tell River whether to retry this job.
	if retryableFailures > 0 {
		return fmt.Errorf("webhook delivery had %d retryable failures", retryableFailures)
	}
	return nil
}

// sendSingleWebhook sends the webhook request once and logs the result.
// Returns (shouldRetry, err). `shouldRetry=true` only for retryable cases (network errors, 5xx).
func (w *Worker) sendSingleWebhook(ctx context.Context, ep WebhookEndpoint, eventID string,
	payload []byte, attempt int) (bool, error) {
	// Compute signature (HMAC SHA-256).
	sig := computeWebhookSignature([]byte(ep.Secret), payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, bytes.NewReader(payload))
	if err != nil {
		// Bad URL or request build error -> non-retryable.
		w.logDelivery(ctx, eventID, ep.ID, "non_retryable_error", attempt, 0, err.Error())
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Ledger-Signature", sig)
	req.Header.Set("User-Agent", "LedgerKiro-Webhook/1.0")

	resp, err := w.HttpClient.Do(req)

	status := "success"
	httpStatus := 0
	errorMessage := ""
	shouldRetry := false

	if err != nil {
		// Network/timeout/DNS errors -> retryable.
		status = "retryable_error"
		errorMessage = err.Error()
		shouldRetry = true
	} else {
		httpStatus = resp.StatusCode

		// Always fully read+close response body to allow connection reuse.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		// Decide retry policy based on HTTP status.
		if resp.StatusCode >= 500 {
			status = "retryable_error"
			errorMessage = fmt.Sprintf("server error: %d", resp.StatusCode)
			shouldRetry = true
		} else if resp.StatusCode >= 400 {
			// 4xx typically indicates a bad endpoint config/auth; do not retry forever.
			status = "non_retryable_error"
			errorMessage = fmt.Sprintf("client error: %d", resp.StatusCode)
			shouldRetry = false
		}
	}

	// Persist delivery attempt.
	w.logDelivery(ctx, eventID, ep.ID, status, attempt, httpStatus, errorMessage)

	if shouldRetry {
		return true, fmt.Errorf("retryable failure for %s: %s", ep.URL, errorMessage)
	}
	return false, nil
}

// logDelivery writes one delivery attempt row.
// Note: errors are intentionally ignored here to avoid masking webhook send results.
func (w *Worker) logDelivery(ctx context.Context, eventID, endpointID, status string, attempt, httpStatus int, errorMessage string) {
	_, _ = w.DB.Exec(ctx, `
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
