package ledger

import (
	"Go_FormanceLegder/internal/api"
	"Go_FormanceLegder/internal/auth"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type EventResponse struct {
	ID            string                 `json:"id"`
	AggregateType string                 `json:"aggregate_type"`
	AggregateID   string                 `json:"aggregate_id"`
	EventType     string                 `json:"event_type"`
	Payload       map[string]interface{} `json:"payload"`
	OccurredAt    string                 `json:"occurred_at"`
	CreatedAt     string                 `json:"created_at"`
}

type ListEventsResponse struct {
	Events     []EventResponse        `json:"events"`
	Pagination api.PaginationResponse `json:"pagination"`
}

// GET /v1/events - List events with pagination
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	limit = api.ValidateLimit(limit)

	continuationToken := r.URL.Query().Get("continuation_token")
	cursor, err := api.DecodeCursor(continuationToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse filters
	eventType := r.URL.Query().Get("event_type")
	aggregateID := r.URL.Query().Get("aggregate_id")

	// Build query
	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, occurred_at, created_at
		FROM events
		WHERE ledger_id = $1
	`
	args := []interface{}{principal.LedgerID}
	argCount := 1

	// Add cursor condition
	if cursor.Timestamp.IsZero() == false {
		argCount++
		query += ` AND (created_at, id) < ($` + fmt.Sprintf("%d", argCount) + `, $` + fmt.Sprintf("%d", argCount+1) + `)`
		args = append(args, cursor.Timestamp, cursor.ID)
		argCount++
	}

	// Add filters
	if eventType != "" {
		argCount++
		query += ` AND event_type = $` + fmt.Sprintf("%d", argCount)
		args = append(args, eventType)
	}
	if aggregateID != "" {
		argCount++
		query += ` AND aggregate_id = $` + fmt.Sprintf("%d", argCount)
		args = append(args, aggregateID)
	}

	// Order and limit
	query += ` ORDER BY created_at DESC, id DESC LIMIT $` + fmt.Sprintf("%d", argCount+1)
	args = append(args, limit+1)

	rows, err := h.Service.DB.Query(ctx, query, args...)
	if err != nil {
		http.Error(w, "failed to query events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	events := []EventResponse{}
	var lastCreatedAt time.Time
	var lastID string

	for rows.Next() {
		var evt EventResponse
		var createdAt, occurredAt time.Time
		var payloadJSON []byte

		err = rows.Scan(&evt.ID, &evt.AggregateType, &evt.AggregateID, &evt.EventType, &payloadJSON, &occurredAt, &createdAt)
		if err != nil {
			http.Error(w, "failed to scan event", http.StatusInternalServerError)
			return
		}

		if err := json.Unmarshal(payloadJSON, &evt.Payload); err != nil {
			http.Error(w, "failed to parse event payload", http.StatusInternalServerError)
			return
		}

		evt.OccurredAt = occurredAt.Format(time.RFC3339)
		evt.CreatedAt = createdAt.Format(time.RFC3339)

		// Stop if we've reached the limit
		if len(events) >= limit {
			break
		}

		events = append(events, evt)
		lastCreatedAt = createdAt
		lastID = evt.ID
	}

	// Check if there are more results
	hasMore := false
	if err = rows.Err(); err == nil {
		if rows.Next() {
			hasMore = true
		}
	}

	// Generate continuation token
	var nextToken string
	if hasMore && len(events) > 0 {
		nextCursor := api.Cursor{
			Timestamp: lastCreatedAt,
			ID:        lastID,
		}
		nextToken, _ = api.EncodeCursor(nextCursor)
	}

	response := ListEventsResponse{
		Events: events,
		Pagination: api.PaginationResponse{
			HasMore:           hasMore,
			ContinuationToken: nextToken,
			Count:             len(events),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /v1/events/:id - Get a specific event
func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	eventID := r.URL.Query().Get("id")
	if eventID == "" {
		http.Error(w, "event id required", http.StatusBadRequest)
		return
	}

	var evt EventResponse
	var createdAt, occurredAt time.Time
	var payloadJSON []byte

	err = h.Service.DB.QueryRow(ctx, `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, occurred_at, created_at
		FROM events
		WHERE ledger_id = $1 AND id = $2
	`, principal.LedgerID, eventID).Scan(&evt.ID, &evt.AggregateType, &evt.AggregateID, &evt.EventType, &payloadJSON, &occurredAt, &createdAt)
	if err != nil {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}

	if err := json.Unmarshal(payloadJSON, &evt.Payload); err != nil {
		http.Error(w, "failed to parse event payload", http.StatusInternalServerError)
		return
	}

	evt.OccurredAt = occurredAt.Format(time.RFC3339)
	evt.CreatedAt = createdAt.Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(evt)
}
