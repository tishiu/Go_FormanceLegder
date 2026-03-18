package ledger

import (
	"Go_FormanceLegder/internal/api"
	"Go_FormanceLegder/internal/auth"
	"encoding/json"
	"fmt"
	"net/http"
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

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	events, pagination, err := h.scope(principal.LedgerID).ListEvents(ctx, EventListParams{
		Limit:             limit,
		ContinuationToken: r.URL.Query().Get("continuation_token"),
		EventType:         r.URL.Query().Get("event_type"),
		AggregateID:       r.URL.Query().Get("aggregate_id"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := ListEventsResponse{
		Events:     events,
		Pagination: pagination,
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

	evt, err := h.scope(principal.LedgerID).GetEvent(ctx, eventID)
	if err != nil {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(evt)
}
