package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

type PaginationRequest struct {
	Limit             int    `json:"limit"`
	ContinuationToken string `json:"continuation_token,omitempty"`
}

type PaginationResponse struct {
	HasMore           bool   `json:"has_more"`
	ContinuationToken string `json:"continuation_token,omitempty"`
	Count             int    `json:"count"`
}

type Cursor struct {
	Timestamp time.Time `json:"timestamp"`
	ID        string    `json:"id"`
}

func EncodeCursor(cursor Cursor) (string, error) {
	data, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

func DecodeCursor(token string) (Cursor, error) {
	var cursor Cursor
	if token == "" {
		return cursor, nil
	}

	data, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return cursor, fmt.Errorf("invalid continuation token")
	}

	if err := json.Unmarshal(data, &cursor); err != nil {
		return cursor, fmt.Errorf("invalid continuation token")
	}

	return cursor, nil
}

func ValidateLimit(limit int) int {
	if limit <= 0 {
		return 100 // default
	}
	if limit > 1000 {
		return 1000 // max
	}
	return limit
}
