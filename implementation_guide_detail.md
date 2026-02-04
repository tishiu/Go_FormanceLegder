# LaaS Implementation Guide — Detailed Step-by-Step

## 1. Introduction

This document provides a complete, step-by-step guide to implementing the Ledger-as-a-Service (LaaS) platform from scratch. It expands on the concepts in the overview document with concrete implementation details, code examples, and configuration steps.

By following this guide, you will build a working multi-tenant ledger system with:
- Secure authentication and authorization
- Double-entry transaction posting with CQRS and event sourcing
- Reliable webhook delivery using transactional outbox
- A React-based developer dashboard

## 2. Prerequisites and Environment Setup

### 2.1 Required Software

Install the following tools:

- **Go 1.22 or later**: Download from https://go.dev/dl/
- **Postgres 14 or later**: Install via package manager or Docker
- **Node.js 18 or later**: For the React frontend
- **golang-migrate**: For database migrations
  ```bash
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  ```

### 2.2 Project Structure

Create the following directory structure:

```
ledger/
├── cmd/
│   ├── api/          # API server entry point
│   └── worker/       # Background worker entry point
├── internal/
│   ├── auth/         # Authentication middleware
│   ├── ledger/       # Ledger service and domain logic
│   ├── projector/    # Event projector
│   ├── webhook/      # Webhook worker
│   └── config/       # Configuration loading
├── migrations/       # SQL migration files
├── web/              # React frontend
│   ├── src/
│   │   ├── api/      # API client and React Query hooks
│   │   ├── features/ # Feature-based components
│   │   └── main.tsx  # Entry point
│   └── package.json
├── go.mod
└── README.md
```

### 2.3 Initialize Go Module

```bash
mkdir ledger && cd ledger
go mod init github.com/yourorg/ledger
```

Install core dependencies:

```bash
go get github.com/jackc/pgx/v5/pgxpool
go get github.com/jackc/pgx/v5
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto/bcrypt
go get github.com/riverqueue/river
go get github.com/riverqueue/river/riverdriver/riverpgxv5
go get github.com/google/uuid
```


## 3. Phase 1 — Database and IAM Foundation

### 3.1 Database Setup

Start a Postgres instance using Docker:

```bash
docker run --name ledger-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=ledger_kiro \
  -p 5432:5432 \
  -d postgres:16
```

Or install Postgres locally and create the database:

```sql
CREATE DATABASE ledger_kiro;
```

### 3.2 Create IAM Migrations

Create `migrations/000001_create_iam_tables.up.sql`:

```sql
-- Users table
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Organizations table
CREATE TABLE organizations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Organization users junction table
CREATE TABLE org_users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL CHECK (role IN ('owner', 'developer')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (organization_id, user_id)
);

CREATE INDEX idx_org_users_org ON org_users(organization_id);
CREATE INDEX idx_org_users_user ON org_users(user_id);

-- Projects table
CREATE TABLE projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  code TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (organization_id, code)
);

CREATE INDEX idx_projects_org ON projects(organization_id);

-- Ledgers table
CREATE TABLE ledgers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  code TEXT NOT NULL,
  currency TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (project_id, code)
);

CREATE INDEX idx_ledgers_project ON ledgers(project_id);

-- API Keys table
CREATE TABLE api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ledger_id UUID NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
  key_hash TEXT NOT NULL UNIQUE,
  prefix TEXT NOT NULL,
  description TEXT,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_ledger ON api_keys(ledger_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
```

Create `migrations/000001_create_iam_tables.down.sql`:

```sql
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS ledgers;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS org_users;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;
```

Apply migrations:

```bash
migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/ledger_kiro?sslmode=disable" up
```


### 3.3 Configuration Management

Create `internal/config/config.go`:

```go
package config

import (
	"os"
	"time"
)

type Config struct {
	DatabaseURL    string
	ServerPort     string
	JWTSecret      []byte
	APIKeySecret   []byte
	SessionTimeout time.Duration
}

func Load() *Config {
	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ledger_kiro?sslmode=disable"),
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		JWTSecret:      []byte(getEnv("JWT_SECRET", "change-me-in-production")),
		APIKeySecret:   []byte(getEnv("API_KEY_SECRET", "change-me-in-production")),
		SessionTimeout: time.Hour * 24,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
```

### 3.4 Database Connection Pool

Create `internal/db/db.go`:

```go
package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 20
	config.MinConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return pool, nil
}
```

### 3.5 Password Hashing Utilities

Create `internal/auth/password.go`:

```go
package auth

import (
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(raw string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash string, raw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw))
}
```

### 3.6 JWT Token Generation

Create `internal/auth/jwt.go`:

```go
package auth

import (
	"time"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"sub"`
	OrgID  string `json:"org_id"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID, orgID string, ttl time.Duration, secret []byte) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		OrgID:  orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ValidateJWT(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
```


### 3.7 API Key Authentication Middleware

Create `internal/auth/middleware.go`:	

```go
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

		keyHash, err := computeKeyHash(m.APIKeySecret, apiKey)
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

func computeKeyHash(secret []byte, key string) (string, error) {
	h := hmac.New(sha256.New, secret)
	_, err := h.Write([]byte(key))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}
```

### 3.8 Basic API Server

Create `cmd/api/main.go`:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/yourorg/ledger/internal/auth"
	"github.com/yourorg/ledger/internal/config"
	"github.com/yourorg/ledger/internal/db"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// TODO: Add more routes

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: mux,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
```

Test the server:

```bash
go run cmd/api/main.go
# In another terminal:
curl http://localhost:8080/health
```


## 4. Phase 2 — Ledger Core with CQRS and Event Sourcing

### 4.1 Create Ledger Core Migrations

Create `migrations/000002_create_ledger_tables.up.sql`:

```sql
-- Events table (source of truth)
CREATE TABLE events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ledger_id UUID NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
  aggregate_type TEXT NOT NULL,
  aggregate_id UUID NOT NULL,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  idempotency_key TEXT,
  UNIQUE (ledger_id, idempotency_key)
);

CREATE INDEX idx_events_ledger ON events(ledger_id);
CREATE INDEX idx_events_aggregate ON events(aggregate_type, aggregate_id);
CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_created ON events(created_at);

-- Accounts table (read model)
CREATE TABLE accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ledger_id UUID NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL CHECK (type IN ('asset', 'liability', 'equity', 'revenue', 'expense')),
  balance NUMERIC(38, 10) NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (ledger_id, code)
);

CREATE INDEX idx_accounts_ledger_code ON accounts(ledger_id, code);

-- Transactions table (read model)
CREATE TABLE transactions (
  id UUID PRIMARY KEY,
  ledger_id UUID NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
  external_id TEXT,
  amount NUMERIC(38, 10) NOT NULL,
  currency TEXT NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (id, ledger_id)
);

CREATE INDEX idx_transactions_ledger ON transactions(ledger_id);
CREATE INDEX idx_transactions_external ON transactions(ledger_id, external_id);

-- Postings table (read model)
CREATE TABLE postings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ledger_id UUID NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
  transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
  account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  amount NUMERIC(38, 10) NOT NULL,
  direction TEXT NOT NULL CHECK (direction IN ('debit', 'credit')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_postings_ledger ON postings(ledger_id);
CREATE INDEX idx_postings_transaction ON postings(transaction_id);
CREATE INDEX idx_postings_account ON postings(account_id);

-- Projector offsets table
CREATE TABLE projector_offsets (
  projector_name TEXT PRIMARY KEY,
  last_processed_event_id UUID NOT NULL
);
```

Create `migrations/000002_create_ledger_tables.down.sql`:

```sql
DROP TABLE IF EXISTS projector_offsets;
DROP TABLE IF EXISTS postings;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS events;
```

Apply migrations:

```bash
migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/ledger_kiro?sslmode=disable" up
```


### 4.2 Ledger Domain Types

Create `internal/ledger/types.go`:

```go
package ledger

import "time"

type PostingInput struct {
	AccountCode string `json:"account_code"`
	Direction   string `json:"direction"`
	Amount      string `json:"amount"`
}

type PostTransactionCommand struct {
	LedgerID       string
	ExternalID     string
	IdempotencyKey string
	Currency       string
	Postings       []PostingInput
	OccurredAt     time.Time
}

type Account struct {
	ID      string
	Code    string
	Type    string
	Balance string
}
```

### 4.3 Ledger Service Implementation

Create `internal/ledger/service.go`:

```go
package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	DB *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{DB: db}
}

func (s *Service) PostTransaction(ctx context.Context, cmd PostTransactionCommand) (string, error) {
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	// Check idempotency
	var existingID string
	err = tx.QueryRow(ctx, `
		SELECT aggregate_id
		FROM events
		WHERE ledger_id = $1
		  AND idempotency_key = $2
	`, cmd.LedgerID, cmd.IdempotencyKey).Scan(&existingID)
	if err == nil {
		// Already processed
		return existingID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	// Load and lock accounts
	accounts, err := s.loadAndLockAccounts(ctx, tx, cmd.LedgerID, cmd.Postings)
	if err != nil {
		return "", err
	}

	// Validate double-entry
	if err := validateDoubleEntry(cmd, accounts); err != nil {
		return "", err
	}

	// Append event
	eventID := uuid.NewString()
	transactionID := uuid.NewString()

	payload := map[string]any{
		"transaction_id": transactionID,
		"external_id":    cmd.ExternalID,
		"currency":       cmd.Currency,
		"occurred_at":    cmd.OccurredAt.UTC().Format(time.RFC3339Nano),
		"postings":       cmd.Postings,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO events (
			id,
			ledger_id,
			aggregate_type,
			aggregate_id,
			event_type,
			payload,
			occurred_at,
			idempotency_key
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, eventID, cmd.LedgerID, "ledger", transactionID, "TransactionPosted", payloadJSON, cmd.OccurredAt, cmd.IdempotencyKey)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return transactionID, nil
}

func (s *Service) loadAndLockAccounts(ctx context.Context, tx pgx.Tx, ledgerID string, postings []PostingInput) (map[string]Account, error) {
	codesSet := map[string]struct{}{}
	for _, p := range postings {
		codesSet[p.AccountCode] = struct{}{}
	}
	codes := make([]string, 0, len(codesSet))
	for c := range codesSet {
		codes = append(codes, c)
	}
	sort.Strings(codes) // Deterministic lock order

	rows, err := tx.Query(ctx, `
		SELECT id, code, type, balance
		FROM accounts
		WHERE ledger_id = $1
		  AND code = ANY($2)
		FOR UPDATE
	`, ledgerID, codes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := map[string]Account{}
	for rows.Next() {
		var a Account
		err = rows.Scan(&a.ID, &a.Code, &a.Type, &a.Balance)
		if err != nil {
			return nil, err
		}
		accounts[a.Code] = a
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(accounts) != len(codes) {
		return nil, fmt.Errorf("one or more accounts not found for ledger %s", ledgerID)
	}

	return accounts, nil
}
```


### 4.4 Double-Entry Validation

Create `internal/ledger/validation.go`:

```go
package ledger

import (
	"fmt"
	"math/big"
)

func validateDoubleEntry(cmd PostTransactionCommand, accounts map[string]Account) error {
	if len(cmd.Postings) < 2 {
		return fmt.Errorf("transaction must have at least 2 postings")
	}

	// Group by currency and sum debits/credits
	totalDebits := new(big.Rat)
	totalCredits := new(big.Rat)

	for _, p := range cmd.Postings {
		// Verify account exists
		if _, ok := accounts[p.AccountCode]; !ok {
			return fmt.Errorf("account %s not found", p.AccountCode)
		}

		// Verify direction
		if p.Direction != "debit" && p.Direction != "credit" {
			return fmt.Errorf("invalid direction: %s", p.Direction)
		}

		// Parse amount
		amount := new(big.Rat)
		if _, ok := amount.SetString(p.Amount); !ok {
			return fmt.Errorf("invalid amount: %s", p.Amount)
		}

		// Check positive
		if amount.Sign() <= 0 {
			return fmt.Errorf("amount must be positive: %s", p.Amount)
		}

		// Accumulate
		if p.Direction == "debit" {
			totalDebits.Add(totalDebits, amount)
		} else {
			totalCredits.Add(totalCredits, amount)
		}
	}

	// Verify balance
	if totalDebits.Cmp(totalCredits) != 0 {
		return fmt.Errorf("debits (%s) must equal credits (%s)", totalDebits.FloatString(10), totalCredits.FloatString(10))
	}

	return nil
}
```

### 4.5 PostTransaction HTTP Handler

Create `internal/ledger/handler.go`:

```go
package ledger

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/yourorg/ledger/internal/auth"
)

type Handler struct {
	Service *Service
}

type PostTransactionRequest struct {
	IdempotencyKey string         `json:"idempotency_key"`
	ExternalID     string         `json:"external_id"`
	Currency       string         `json:"currency"`
	OccurredAt     time.Time      `json:"occurred_at"`
	Postings       []PostingInput `json:"postings"`
}

type PostTransactionResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

func (h *Handler) PostTransaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, err := auth.FromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req PostTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	cmd := PostTransactionCommand{
		LedgerID:       principal.LedgerID,
		ExternalID:     req.ExternalID,
		IdempotencyKey: req.IdempotencyKey,
		Currency:       req.Currency,
		OccurredAt:     req.OccurredAt,
		Postings:       req.Postings,
	}

	transactionID, err := h.Service.PostTransaction(ctx, cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := PostTransactionResponse{
		TransactionID: transactionID,
		Status:        "accepted",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
```


### 4.6 Projector Implementation

Create `internal/projector/projector.go`:

```go
package projector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Projector struct {
	DB *pgxpool.Pool
}

func NewProjector(db *pgxpool.Pool) *Projector {
	return &Projector{DB: db}
}

func (p *Projector) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := p.projectBatch(ctx); err != nil {
				log.Printf("projection error: %v", err)
			}
		}
	}
}

func (p *Projector) projectBatch(ctx context.Context) error {
	tx, err := p.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Load Events
	type EventData struct {
		ID, LedgerID, Type string
		Payload            []byte
	}
	var events []EventData

	rows, err := tx.Query(ctx, `
       SELECT id, ledger_id, event_type, payload
       FROM events
       WHERE event_type = 'TransactionPosted'
         AND id > COALESCE((SELECT last_processed_event_id FROM projector_offsets WHERE projector_name = 'ledger'), '00000000-0000-0000-0000-000000000000')
       ORDER BY created_at, id
       LIMIT 100
    `)
	if err != nil {
		return err
	}
	for rows.Next() {
		var e EventData
		if err := rows.Scan(&e.ID, &e.LedgerID, &e.Type, &e.Payload); err != nil {
			rows.Close() // Nhớ close nếu return sớm
			return err
		}
		events = append(events, e)
	}
	rows.Close()

	if len(events) == 0 {
		return tx.Commit(ctx)
	}

	// Process
	var maxEventID string
	for _, event := range events {
		var payload map[string]any
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("bad payload event %s: %w", event.ID, err)
		}

		// Pass tx xuống để xử lý
		if err := p.applyTransactionPosted(ctx, tx, event.LedgerID, payload); err != nil {
			return fmt.Errorf("failed apply event %s: %w", event.ID, err)
		}
		maxEventID = event.ID
	}

	// Update Offset
	_, err = tx.Exec(ctx, `
       INSERT INTO projector_offsets (projector_name, last_processed_event_id)
       VALUES ('ledger', $1)
       ON CONFLICT (projector_name)
       DO UPDATE SET last_processed_event_id = EXCLUDED.last_processed_event_id
    `, maxEventID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (p *Projector) applyTransactionPosted(ctx context.Context, tx pgx.Tx, ledgerID string, payload map[string]any) error {
	transactionID := payload["transaction_id"].(string)
	externalID, _ := payload["external_id"].(string)
	currency := payload["currency"].(string)
	occurredAtStr := payload["occurred_at"].(string)
	occurredAt, err := time.Parse(time.RFC3339Nano, occurredAtStr)
	if err != nil {
		return fmt.Errorf("invalid time format: %w", err)
	}

	// Insert transaction
	// tag.RowsAffected() == 1: Insert successful
	// tag.RowsAffected() == 0: (Old Transaction) -> RETURN
	tag, err := tx.Exec(ctx, `
       INSERT INTO transactions (
          id, ledger_id, external_id, amount, currency, occurred_at
       ) VALUES ($1, $2, $3, $4, $5, $6)
       ON CONFLICT (id, ledger_id) DO NOTHING
    `, transactionID, ledgerID, externalID, "0", currency, occurredAt)
	if err != nil {
		return fmt.Errorf("insert transaction failed: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return nil
	}

	// Process postings
	postings, ok := payload["postings"].([]any)
	if !ok {
		return fmt.Errorf("invalid postings payload")
	}

	for _, raw := range postings {
		pMap := raw.(map[string]any)
		accountCode := pMap["account_code"].(string)
		direction := pMap["direction"].(string)
		amount := pMap["amount"].(string)

		// TODO: Find AccountID, using cache if possible
		var accountID string
		err = tx.QueryRow(ctx, `
          SELECT id FROM accounts WHERE ledger_id = $1 AND code = $2
       `, ledgerID, accountCode).Scan(&accountID)

		if err != nil {
			return fmt.Errorf("account %s not found: %w", accountCode, err)
		}

		// Persist Posting Log
		postingID := uuid.NewString()
		_, err = tx.Exec(ctx, `
			INSERT INTO postings (
				id,
				ledger_id,
				transaction_id,
				account_id,
				amount,
				direction
			) VALUES ($1, $2, $3, $4, $5, $6)
		`, postingID, ledgerID, transactionID, accountID, amount, direction)
		if err != nil {
			return fmt.Errorf("insert posting failed: %w", err)
		}

		// Update account balance
		if err := p.updateAccountBalance(ctx, tx, accountID, direction, amount); err != nil {
			return err
		}
	}

	return nil
}

func (p *Projector) updateAccountBalance(ctx context.Context, tx pgx.Tx, accountID, direction, amountStr string) error {
	amount := new(big.Rat)
	if _, ok := amount.SetString(amountStr); !ok {
		return fmt.Errorf("invalid amount: %s", amountStr)
	}

	var finalAmount *big.Rat
	if direction == "credit" {
		finalAmount = amount
	} else {
		finalAmount = new(big.Rat).Neg(amount)
	}

	_, err := tx.Exec(ctx, `
       UPDATE accounts 
       SET balance = balance + $1 
       WHERE id = $2
    `, finalAmount.FloatString(10), accountID)

	return err
}
```


### 4.7 Worker Entry Point

Create `cmd/worker/main.go`:

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/yourorg/ledger/internal/config"
	"github.com/yourorg/ledger/internal/db"
	"github.com/yourorg/ledger/internal/projector"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	proj := projector.NewProjector(pool)

	go func() {
		log.Println("Projector worker starting...")
		if err := proj.Run(ctx); err != nil {
			log.Printf("projector error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down worker...")
	cancel()
	log.Println("Worker stopped")
}
```

Test the projector:

```bash
# Terminal 1: Start API
go run cmd/api/main.go

# Terminal 2: Start worker
go run cmd/worker/main.go

# Terminal 3: Post a transaction (after creating accounts)
# See testing section below
```


## 5. Phase 3 — Webhook Engine with River

### 5.1 Create Webhook Migrations

Create `migrations/000003_create_webhook_tables.up.sql`:

```sql
-- Webhook endpoints table
CREATE TABLE webhook_endpoints (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ledger_id UUID NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
  url TEXT NOT NULL,
  secret TEXT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_endpoints_ledger ON webhook_endpoints(ledger_id);

-- Webhook deliveries table
CREATE TABLE webhook_deliveries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  webhook_endpoint_id UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
  status TEXT NOT NULL CHECK (status IN ('success', 'retryable_error', 'non_retryable_error')),
  attempt INT NOT NULL,
  last_attempt_at TIMESTAMPTZ,
  http_status INT,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_event ON webhook_deliveries(event_id);
CREATE INDEX idx_webhook_deliveries_endpoint ON webhook_deliveries(webhook_endpoint_id);
```

Create `migrations/000003_create_webhook_tables.down.sql`:

```sql
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhook_endpoints;
```

Apply migrations:

```bash
migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/ledger_kiro?sslmode=disable" up
```

### 5.2 Install River

River requires its own migrations. Create them:

```bash
# Install River CLI
go install github.com/riverqueue/river/cmd/river@latest

# Generate River migrations
river migrate-generate --database-url "postgres://postgres:postgres@localhost:5432/ledger_kiro?sslmode=disable"
```

This creates River's job tables. Alternatively, manually create `migrations/000004_create_river_tables.up.sql` with River's schema (see River documentation).

### 5.3 Webhook Worker Types

Create `internal/webhook/types.go`:

```go
package webhook

type WebhookArgs struct {
	EventID  string `json:"event_id"`
	LedgerID string `json:"ledger_id"`
}

func (WebhookArgs) Kind() string {
	return "webhook_delivery"
}
```


### 5.4 Webhook Worker Implementation

Create `internal/webhook/worker.go`:

```go
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
```

### 5.5 Integrate River with Ledger Service

Update `internal/ledger/service.go` to enqueue webhook jobs:

```go
// Add River client to Service struct
import (
	"github.com/riverqueue/river"
	"github.com/yourorg/ledger/internal/webhook"
)

type Service struct {
	DB          *pgxpool.Pool
	RiverClient *river.Client[pgx.Tx]
}

// Update PostTransaction to enqueue webhook job
func (s *Service) PostTransaction(ctx context.Context, cmd PostTransactionCommand) (string, error) {
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	// ... existing idempotency check ...
	// ... existing validation ...
	// ... existing event append ...

	// Enqueue webhook job atomically
	_, err = s.RiverClient.InsertTx(ctx, tx, webhook.WebhookArgs{
		EventID:  eventID,
		LedgerID: cmd.LedgerID,
	}, nil)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return transactionID, nil
}
```


### 5.6 Update Worker to Include River

Update `cmd/worker/main.go`:

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/yourorg/ledger/internal/config"
	"github.com/yourorg/ledger/internal/db"
	"github.com/yourorg/ledger/internal/projector"
	"github.com/yourorg/ledger/internal/webhook"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Setup River workers
	workers := river.NewWorkers()
	river.AddWorker(workers, &webhook.Worker{DB: pool})

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Workers: workers,
	})
	if err != nil {
		log.Fatalf("failed to create river client: %v", err)
	}

	// Start River
	if err := riverClient.Start(ctx); err != nil {
		log.Fatalf("failed to start river: %v", err)
	}

	// Start projector
	proj := projector.NewProjector(pool)
	go func() {
		log.Println("Projector worker starting...")
		if err := proj.Run(ctx); err != nil {
			log.Printf("projector error: %v", err)
		}
	}()

	log.Println("Worker processes started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down workers...")
	cancel()
	riverClient.Stop(ctx)
	log.Println("Workers stopped")
}
```

### 5.7 Update API Server with River Insert Client

Update `cmd/api/main.go` to create an insert-only River client:

```go
import (
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/yourorg/ledger/internal/ledger"
	"github.com/yourorg/ledger/internal/webhook"
)

func main() {
	// ... existing setup ...

	// Create River insert-only client
	workers := river.NewWorkers()
	river.AddWorker(workers, &webhook.Worker{DB: pool})

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Workers: workers,
	})
	if err != nil {
		log.Fatalf("failed to create river client: %v", err)
	}

	// Create ledger service with River client
	ledgerService := &ledger.Service{
		DB:          pool,
		RiverClient: riverClient,
	}

	ledgerHandler := &ledger.Handler{
		Service: ledgerService,
	}

	// Setup auth middleware
	authMiddleware := &auth.Middleware{
		DB:           pool,
		APIKeySecret: cfg.APIKeySecret,
	}

	// Routes
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.Handle("/v1/transactions", authMiddleware.AuthMiddleware(
		http.HandlerFunc(ledgerHandler.PostTransaction),
	))

	// ... rest of server setup ...
}
```


## 6. Phase 4 — React Dashboard

### 6.1 Initialize React Project

```bash
cd web
npm create vite@latest . -- --template react-ts
npm install
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
npm install @tanstack/react-query axios react-router-dom
```

Configure Tailwind in `web/tailwind.config.js`:

```js
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
```

Update `web/src/index.css`:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

### 6.2 API Client Setup

Create `web/src/api/client.ts`:

```typescript
import axios from "axios";

export const api = axios.create({
  baseURL: "/api",
  withCredentials: true,
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      window.location.href = "/login";
    }
    return Promise.reject(error);
  }
);
```

### 6.3 React Query Setup

Update `web/src/main.tsx`:

```typescript
import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter } from "react-router-dom";
import App from "./App";
import "./index.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </QueryClientProvider>
  </React.StrictMode>
);
```

### 6.4 Login Page

Create `web/src/features/auth/LoginPage.tsx`:

```typescript
import { useState } from "react";
import { api } from "../../api/client";
import { useNavigate } from "react-router-dom";

export function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await api.post("/auth/login", { email, password });
      navigate("/");
    } catch {
      setError("Invalid credentials");
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <form
        onSubmit={onSubmit}
        className="w-full max-w-sm space-y-4 rounded bg-white p-6 shadow"
      >
        <h1 className="text-lg font-semibold">Sign in to LaaS</h1>
        {error && <div className="text-sm text-red-600">{error}</div>}

        <div>
          <label className="block text-sm font-medium">Email</label>
          <input
            className="mt-1 block w-full rounded border px-3 py-2"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium">Password</label>
          <input
            className="mt-1 block w-full rounded border px-3 py-2"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </div>
        <button
          type="submit"
          className="w-full rounded bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
        >
          Sign in
        </button>
      </form>
    </div>
  );
}
```


### 6.5 Ledger Management

Create `web/src/api/ledgers.ts`:

```typescript
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "./client";

export interface Ledger {
  id: string;
  project_id: string;
  name: string;
  code: string;
  currency: string;
  created_at: string;
}

export function useLedgers() {
  return useQuery({
    queryKey: ["ledgers"],
    queryFn: async () => {
      const res = await api.get<Ledger[]>("/ledgers");
      return res.data;
    },
  });
}

export function useCreateLedger() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: {
      project_id: string;
      name: string;
      code: string;
      currency: string;
    }) => {
      const res = await api.post<Ledger>("/ledgers", input);
      return res.data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["ledgers"] });
    },
  });
}
```

Create `web/src/features/ledgers/LedgersPage.tsx`:

```typescript
import { useState } from "react";
import { useLedgers, useCreateLedger } from "../../api/ledgers";

export function LedgersPage() {
  const ledgersQuery = useLedgers();
  const createLedger = useCreateLedger();
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [currency, setCurrency] = useState("USD");

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    createLedger.mutate(
      {
        project_id: "default-project-id", // TODO: Get from context
        name,
        code,
        currency,
      },
      {
        onSuccess: () => {
          setShowForm(false);
          setName("");
          setCode("");
        },
      }
    );
  }

  if (ledgersQuery.isLoading) {
    return <div className="p-6">Loading ledgers...</div>;
  }

  return (
    <div className="p-6">
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Ledgers</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="rounded bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
        >
          Create Ledger
        </button>
      </div>

      {showForm && (
        <form onSubmit={onSubmit} className="mb-6 space-y-4 rounded border p-4">
          <div>
            <label className="block text-sm font-medium">Name</label>
            <input
              className="mt-1 block w-full rounded border px-3 py-2"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Production"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium">Code</label>
            <input
              className="mt-1 block w-full rounded border px-3 py-2"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="production"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium">Currency</label>
            <input
              className="mt-1 block w-full rounded border px-3 py-2"
              value={currency}
              onChange={(e) => setCurrency(e.target.value)}
              placeholder="USD"
              required
            />
          </div>
          <div className="flex gap-2">
            <button
              type="submit"
              className="rounded bg-green-600 px-4 py-2 text-white hover:bg-green-700"
              disabled={createLedger.isPending}
            >
              Create
            </button>
            <button
              type="button"
              onClick={() => setShowForm(false)}
              className="rounded border px-4 py-2 hover:bg-gray-50"
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      <table className="min-w-full border">
        <thead className="bg-gray-50">
          <tr>
            <th className="border px-4 py-2 text-left">Name</th>
            <th className="border px-4 py-2 text-left">Code</th>
            <th className="border px-4 py-2 text-left">Currency</th>
            <th className="border px-4 py-2 text-left">Created</th>
          </tr>
        </thead>
        <tbody>
          {ledgersQuery.data?.map((ledger) => (
            <tr key={ledger.id} className="hover:bg-gray-50">
              <td className="border px-4 py-2">{ledger.name}</td>
              <td className="border px-4 py-2">{ledger.code}</td>
              <td className="border px-4 py-2">{ledger.currency}</td>
              <td className="border px-4 py-2">
                {new Date(ledger.created_at).toLocaleDateString()}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```


### 6.6 API Key Management

Create `web/src/features/api-keys/ApiKeysPage.tsx`:

```typescript
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";

interface ApiKey {
  id: string;
  prefix: string;
  description: string;
  is_active: boolean;
  created_at: string;
}

export function ApiKeysPage({ ledgerID }: { ledgerID: string }) {
  const qc = useQueryClient();
  const [description, setDescription] = useState("");
  const [newKey, setNewKey] = useState<string | null>(null);

  const keysQuery = useQuery({
    queryKey: ["api-keys", ledgerID],
    queryFn: async () => {
      const res = await api.get<ApiKey[]>(`/ledgers/${ledgerID}/api-keys`);
      return res.data;
    },
  });

  const createKey = useMutation({
    mutationFn: async (desc: string) => {
      const res = await api.post<{ raw_key: string }>(`/ledgers/${ledgerID}/api-keys`, {
        description: desc,
      });
      return res.data;
    },
    onSuccess: (data) => {
      setNewKey(data.raw_key);
      qc.invalidateQueries({ queryKey: ["api-keys", ledgerID] });
    },
  });

  const revokeKey = useMutation({
    mutationFn: async (id: string) => {
      await api.post(`/api-keys/${id}/revoke`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["api-keys", ledgerID] });
    },
  });

  if (keysQuery.isLoading) {
    return <div className="p-6">Loading API keys...</div>;
  }

  return (
    <div className="p-6">
      <h1 className="mb-4 text-2xl font-bold">API Keys</h1>

      <form
        className="mb-6 flex items-end gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          createKey.mutate(description);
          setDescription("");
        }}
      >
        <div className="flex-1">
          <label className="block text-sm font-medium">Description</label>
          <input
            className="mt-1 block w-full rounded border px-3 py-2"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Production API key"
            required
          />
        </div>
        <button
          type="submit"
          className="rounded bg-green-600 px-4 py-2 text-white hover:bg-green-700"
          disabled={createKey.isPending}
        >
          Generate Key
        </button>
      </form>

      {newKey && (
        <div className="mb-6 rounded border border-yellow-400 bg-yellow-50 p-4">
          <p className="font-semibold text-yellow-900">
            Copy this key now. You will not see it again.
          </p>
          <pre className="mt-2 overflow-x-auto rounded bg-gray-900 p-2 text-sm text-green-300">
            {newKey}
          </pre>
          <button
            onClick={() => setNewKey(null)}
            className="mt-2 text-sm text-yellow-900 underline"
          >
            Dismiss
          </button>
        </div>
      )}

      <table className="min-w-full border">
        <thead className="bg-gray-50">
          <tr>
            <th className="border px-4 py-2 text-left">Prefix</th>
            <th className="border px-4 py-2 text-left">Description</th>
            <th className="border px-4 py-2 text-left">Status</th>
            <th className="border px-4 py-2 text-left">Created</th>
            <th className="border px-4 py-2"></th>
          </tr>
        </thead>
        <tbody>
          {keysQuery.data?.map((key) => (
            <tr key={key.id} className="hover:bg-gray-50">
              <td className="border px-4 py-2 font-mono text-sm">{key.prefix}</td>
              <td className="border px-4 py-2">{key.description}</td>
              <td className="border px-4 py-2">
                {key.is_active ? (
                  <span className="text-green-600">Active</span>
                ) : (
                  <span className="text-red-600">Revoked</span>
                )}
              </td>
              <td className="border px-4 py-2">
                {new Date(key.created_at).toLocaleDateString()}
              </td>
              <td className="border px-4 py-2 text-right">
                {key.is_active && (
                  <button
                    onClick={() => {
                      if (confirm("Revoke this API key?")) {
                        revokeKey.mutate(key.id);
                      }
                    }}
                    className="text-red-600 hover:underline"
                  >
                    Revoke
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```


### 6.7 Accounts and Transactions Explorer

Create `web/src/features/ledger/AccountsTable.tsx`:

```typescript
import { useQuery } from "@tanstack/react-query";
import { api } from "../../api/client";

interface Account {
  id: string;
  code: string;
  name: string;
  type: string;
  balance: string;
}

export function AccountsTable({ ledgerID }: { ledgerID: string }) {
  const query = useQuery({
    queryKey: ["accounts", ledgerID],
    queryFn: async () => {
      const res = await api.get<Account[]>(`/ledgers/${ledgerID}/accounts`);
      return res.data;
    },
  });

  if (query.isLoading) {
    return <div className="p-6">Loading accounts...</div>;
  }

  return (
    <div className="p-6">
      <h1 className="mb-4 text-2xl font-bold">Accounts</h1>
      <table className="min-w-full border">
        <thead className="bg-gray-50">
          <tr>
            <th className="border px-4 py-2 text-left">Code</th>
            <th className="border px-4 py-2 text-left">Name</th>
            <th className="border px-4 py-2 text-left">Type</th>
            <th className="border px-4 py-2 text-right">Balance</th>
          </tr>
        </thead>
        <tbody>
          {query.data?.map((account) => (
            <tr key={account.id} className="hover:bg-gray-50">
              <td className="border px-4 py-2 font-mono">{account.code}</td>
              <td className="border px-4 py-2">{account.name}</td>
              <td className="border px-4 py-2 capitalize">{account.type}</td>
              <td className="border px-4 py-2 text-right font-mono">
                {parseFloat(account.balance).toFixed(2)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```

Create `web/src/features/webhooks/WebhookLogsTable.tsx`:

```typescript
import { useQuery } from "@tanstack/react-query";
import { api } from "../../api/client";

interface WebhookDelivery {
  id: string;
  event_id: string;
  endpoint_url: string;
  status: string;
  http_status: number;
  attempt: number;
  last_attempt_at: string;
  error_message?: string;
}

export function WebhookLogsTable({ ledgerID }: { ledgerID: string }) {
  const query = useQuery({
    queryKey: ["webhook-logs", ledgerID],
    queryFn: async () => {
      const res = await api.get<WebhookDelivery[]>(`/ledgers/${ledgerID}/webhook-deliveries`);
      return res.data;
    },
  });

  if (query.isLoading) {
    return <div className="p-6">Loading webhook logs...</div>;
  }

  return (
    <div className="p-6">
      <h1 className="mb-4 text-2xl font-bold">Webhook Logs</h1>
      <table className="min-w-full border text-sm">
        <thead className="bg-gray-50">
          <tr>
            <th className="border px-4 py-2 text-left">Event ID</th>
            <th className="border px-4 py-2 text-left">Endpoint</th>
            <th className="border px-4 py-2 text-left">Status</th>
            <th className="border px-4 py-2 text-left">HTTP</th>
            <th className="border px-4 py-2 text-left">Attempt</th>
            <th className="border px-4 py-2 text-left">Last Attempt</th>
          </tr>
        </thead>
        <tbody>
          {query.data?.map((log) => (
            <tr key={log.id} className="hover:bg-gray-50">
              <td className="border px-4 py-2 font-mono text-xs">{log.event_id.slice(0, 8)}</td>
              <td className="border px-4 py-2">{log.endpoint_url}</td>
              <td className="border px-4 py-2">
                <span
                  className={
                    log.status === "success"
                      ? "text-green-600"
                      : log.status === "retryable_error"
                      ? "text-yellow-600"
                      : "text-red-600"
                  }
                >
                  {log.status}
                </span>
              </td>
              <td className="border px-4 py-2">{log.http_status || "-"}</td>
              <td className="border px-4 py-2">{log.attempt}</td>
              <td className="border px-4 py-2">
                {new Date(log.last_attempt_at).toLocaleString()}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```

### 6.8 App Router

Create `web/src/App.tsx`:

```typescript
import { Routes, Route, Navigate } from "react-router-dom";
import { LoginPage } from "./features/auth/LoginPage";
import { LedgersPage } from "./features/ledgers/LedgersPage";
import { ApiKeysPage } from "./features/api-keys/ApiKeysPage";
import { AccountsTable } from "./features/ledger/AccountsTable";
import { WebhookLogsTable } from "./features/webhooks/WebhookLogsTable";

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/ledgers" element={<LedgersPage />} />
      <Route path="/ledgers/:ledgerID/api-keys" element={<ApiKeysPage ledgerID="TODO" />} />
      <Route path="/ledgers/:ledgerID/accounts" element={<AccountsTable ledgerID="TODO" />} />
      <Route path="/ledgers/:ledgerID/webhooks" element={<WebhookLogsTable ledgerID="TODO" />} />
      <Route path="/" element={<Navigate to="/ledgers" replace />} />
    </Routes>
  );
}

export default App;
```


## 7. Testing and Validation

### 7.1 Seed Test Data

Create a seed script to populate test data. Create `scripts/seed.sql`:

```sql
-- Create test user
INSERT INTO users (id, email, password_hash)
VALUES (
  '00000000-0000-0000-0000-000000000001',
  'test@example.com',
  '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy' -- password: "password"
);

-- Create test organization
INSERT INTO organizations (id, name)
VALUES ('00000000-0000-0000-0000-000000000002', 'Test Org');

-- Link user to organization
INSERT INTO org_users (id, organization_id, user_id, role)
VALUES (
  '00000000-0000-0000-0000-000000000003',
  '00000000-0000-0000-0000-000000000002',
  '00000000-0000-0000-0000-000000000001',
  'owner'
);

-- Create test project
INSERT INTO projects (id, organization_id, name, code)
VALUES (
  '00000000-0000-0000-0000-000000000004',
  '00000000-0000-0000-0000-000000000002',
  'Test Project',
  'test'
);

-- Create test ledger
INSERT INTO ledgers (id, project_id, name, code, currency)
VALUES (
  '00000000-0000-0000-0000-000000000005',
  '00000000-0000-0000-0000-000000000004',
  'Sandbox',
  'sandbox',
  'USD'
);

-- Create test accounts
INSERT INTO accounts (id, ledger_id, code, name, type, balance)
VALUES
  (
    '00000000-0000-0000-0000-000000000006',
    '00000000-0000-0000-0000-000000000005',
    'cash',
    'Cash',
    'asset',
    0
  ),
  (
    '00000000-0000-0000-0000-000000000007',
    '00000000-0000-0000-0000-000000000005',
    'revenue',
    'Revenue',
    'revenue',
    0
  );

-- Create test API key (raw key: sk_test_12345678901234567890123456789012)
-- Hash computed with HMAC-SHA256 using secret "change-me-in-production"
INSERT INTO api_keys (id, ledger_id, key_hash, prefix, description, is_active)
VALUES (
  '00000000-0000-0000-0000-000000000008',
  '00000000-0000-0000-0000-000000000005',
  'computed_hash_here', -- Replace with actual hash
  'sk_test_',
  'Test API Key',
  true
);
```

Apply seed data:

```bash
psql -U postgres -d ledger_kiro -f scripts/seed.sql
```

### 7.2 Manual API Testing

Test the PostTransaction endpoint:

```bash
# Post a transaction
curl -X POST http://localhost:8080/v1/transactions \
  -H "Authorization: Bearer sk_test_12345678901234567890123456789012" \
  -H "Content-Type: application/json" \
  -d '{
    "idempotency_key": "test-txn-001",
    "external_id": "order-123",
    "currency": "USD",
    "occurred_at": "2024-01-01T12:00:00Z",
    "postings": [
      {
        "account_code": "cash",
        "direction": "debit",
        "amount": "100.00"
      },
      {
        "account_code": "revenue",
        "direction": "credit",
        "amount": "100.00"
      }
    ]
  }'
```

Expected response:

```json
{
  "transaction_id": "uuid-here",
  "status": "accepted"
}
```

Verify in database:

```sql
-- Check event was created
SELECT * FROM events WHERE ledger_id = '00000000-0000-0000-0000-000000000005';

-- Wait for projector to run, then check accounts
SELECT code, balance FROM accounts WHERE ledger_id = '00000000-0000-0000-0000-000000000005';
```

Expected balances:
- cash: 100.00
- revenue: 100.00


### 7.3 Integration Test

Create `internal/integration/integration_test.go`:

```go
package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/yourorg/ledger/internal/auth"
	"github.com/yourorg/ledger/internal/ledger"
	"github.com/yourorg/ledger/internal/webhook"
)

func TestPostTransactionEndToEnd(t *testing.T) {
	ctx := context.Background()

	// Setup database
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/ledger_kiro_test?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	// Clean database
	cleanDatabase(t, pool)

	// Setup River
	workers := river.NewWorkers()
	river.AddWorker(workers, &webhook.Worker{DB: pool})

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Workers: workers,
	})
	if err != nil {
		t.Fatalf("failed to create river client: %v", err)
	}

	// Create ledger service
	ledgerService := &ledger.Service{
		DB:          pool,
		RiverClient: riverClient,
	}

	// Seed test data
	seedTestData(t, pool)

	// Post transaction
	cmd := ledger.PostTransactionCommand{
		LedgerID:       "00000000-0000-0000-0000-000000000005",
		ExternalID:     "test-order-123",
		IdempotencyKey: "test-idempotency-001",
		Currency:       "USD",
		OccurredAt:     time.Now(),
		Postings: []ledger.PostingInput{
			{AccountCode: "cash", Direction: "debit", Amount: "100.00"},
			{AccountCode: "revenue", Direction: "credit", Amount: "100.00"},
		},
	}

	transactionID, err := ledgerService.PostTransaction(ctx, cmd)
	if err != nil {
		t.Fatalf("failed to post transaction: %v", err)
	}

	if transactionID == "" {
		t.Fatal("expected transaction ID")
	}

	// Verify event was created
	var eventCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM events WHERE ledger_id = $1
	`, cmd.LedgerID).Scan(&eventCount)
	if err != nil {
		t.Fatalf("failed to query events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("expected 1 event, got %d", eventCount)
	}

	// Verify webhook job was created
	var jobCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM river_job WHERE kind = 'webhook_delivery'
	`).Scan(&jobCount)
	if err != nil {
		t.Fatalf("failed to query jobs: %v", err)
	}
	if jobCount != 1 {
		t.Fatalf("expected 1 job, got %d", jobCount)
	}

	t.Log("Integration test passed!")
}

func cleanDatabase(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		TRUNCATE users, organizations, org_users, projects, ledgers, api_keys,
		         events, accounts, transactions, postings, projector_offsets,
		         webhook_endpoints, webhook_deliveries, river_job CASCADE
	`)
	if err != nil {
		t.Fatalf("failed to clean database: %v", err)
	}
}

func seedTestData(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()

	// Create ledger
	_, err := pool.Exec(ctx, `
		INSERT INTO ledgers (id, project_id, name, code, currency)
		VALUES ('00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000004', 'Test', 'test', 'USD')
	`)
	if err != nil {
		t.Fatalf("failed to seed ledger: %v", err)
	}

	// Create accounts
	_, err = pool.Exec(ctx, `
		INSERT INTO accounts (id, ledger_id, code, name, type, balance)
		VALUES
		  ('00000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000005', 'cash', 'Cash', 'asset', 0),
		  ('00000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000005', 'revenue', 'Revenue', 'revenue', 0)
	`)
	if err != nil {
		t.Fatalf("failed to seed accounts: %v", err)
	}
}
```

Run the test:

```bash
go test ./internal/integration -v
```


## 8. Deployment and Operations

### 8.1 Environment Variables

Create `.env` file for local development:

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5432/ledger_kiro?sslmode=disable
SERVER_PORT=8080
JWT_SECRET=your-jwt-secret-change-in-production
API_KEY_SECRET=your-api-key-secret-change-in-production
```

For production, use a secrets manager (AWS Secrets Manager, HashiCorp Vault, etc.).

### 8.2 Docker Compose Setup

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: ledger_kiro
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  api:
    build:
      context: .
      dockerfile: Dockerfile.api
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres:5432/ledger_kiro?sslmode=disable
      SERVER_PORT: 8080
      JWT_SECRET: ${JWT_SECRET}
      API_KEY_SECRET: ${API_KEY_SECRET}
    depends_on:
      - postgres

  worker:
    build:
      context: .
      dockerfile: Dockerfile.worker
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres:5432/ledger_kiro?sslmode=disable
    depends_on:
      - postgres

volumes:
  postgres_data:
```

Create `Dockerfile.api`:

```dockerfile
FROM golang:1.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/api .
EXPOSE 8080
CMD ["./api"]
```

Create `Dockerfile.worker`:

```dockerfile
FROM golang:1.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/worker .
CMD ["./worker"]
```

Start services:

```bash
docker-compose up -d
```

### 8.3 Database Migrations in Production

Use a migration tool in your CI/CD pipeline:

```bash
# Run migrations before deploying new code
migrate -path ./migrations \
  -database "${DATABASE_URL}" \
  up
```

### 8.4 Monitoring and Observability

Key metrics to track:

**API Metrics:**
- Request rate and latency (p50, p95, p99)
- Error rate by endpoint
- API key authentication failures

**Ledger Metrics:**
- PostTransaction throughput
- Validation error rate
- Event append latency

**Projector Metrics:**
- Projection lag (latest event ID - processed event ID)
- Projection throughput (events/second)
- Projection errors

**Webhook Metrics:**
- Job queue depth
- Delivery success rate
- Retry rate
- Average delivery latency

**Database Metrics:**
- Connection pool utilization
- Query latency
- Lock contention
- Table sizes

Implement structured logging:

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("transaction posted",
	"transaction_id", transactionID,
	"ledger_id", ledgerID,
	"organization_id", orgID,
)
```


### 8.5 Backup and Recovery

**Database Backups:**

```bash
# Daily automated backup
pg_dump -U postgres ledger_kiro | gzip > backup_$(date +%Y%m%d).sql.gz

# Restore from backup
gunzip -c backup_20240101.sql.gz | psql -U postgres ledger_kiro
```

**Event Store Replay:**

If the read model becomes corrupted, rebuild from events:

```sql
-- Clear read models
TRUNCATE accounts, transactions, postings CASCADE;
UPDATE projector_offsets SET last_processed_event_id = '00000000-0000-0000-0000-000000000000';

-- Restart projector worker to replay all events
```

### 8.6 Scaling Considerations

**Vertical Scaling:**
- Start with a single Postgres instance with sufficient CPU and memory
- Use connection pooling (pgBouncer) for high connection counts

**Horizontal Scaling:**
- Add read replicas for dashboard queries
- Scale River workers horizontally by running multiple worker processes
- Partition projectors by ledger_id if needed

**Database Partitioning:**
- Partition events table by created_at for time-based queries
- Partition by ledger_id for tenant isolation

**Caching:**
- Cache account balances with Redis for high-read workloads
- Invalidate cache on transaction posting

## 9. Security Hardening

### 9.1 API Key Security

- Rotate API key secrets periodically
- Implement rate limiting per API key
- Log all API key usage for audit trails
- Support key expiration dates

### 9.2 Database Security

Enable Row-Level Security (RLS):

```sql
ALTER TABLE accounts ENABLE ROW LEVEL SECURITY;

CREATE POLICY ledger_isolation ON accounts
  USING (ledger_id = current_setting('app.ledger_id')::uuid);
```

Set session variable in middleware:

```go
_, err = tx.Exec(ctx, "SELECT set_config('app.ledger_id', $1, true)", ledgerID)
```

### 9.3 Network Security

- Use TLS for all external connections
- Restrict database access to application servers only
- Use VPC/private networks for internal communication
- Implement IP whitelisting for webhook endpoints if needed

### 9.4 Secrets Management

- Never commit secrets to version control
- Use environment variables or secrets managers
- Rotate secrets regularly
- Use different secrets for each environment

## 10. Troubleshooting Guide

### 10.1 Common Issues

**Issue: Projection lag increasing**

Symptoms: Read models are stale, balances not updating

Solutions:
- Check projector worker logs for errors
- Verify database connection pool has capacity
- Scale projector workers horizontally
- Optimize projection queries

**Issue: Webhook delivery failures**

Symptoms: Deliveries stuck in retryable_error state

Solutions:
- Check tenant endpoint availability
- Verify webhook signature validation
- Review error_message in webhook_deliveries table
- Implement circuit breaker for failing endpoints

**Issue: API key authentication failures**

Symptoms: 401 errors on valid API keys

Solutions:
- Verify API_KEY_SECRET matches between key generation and validation
- Check that key is not revoked (is_active = true)
- Verify Authorization header format

**Issue: Double-entry validation errors**

Symptoms: Transactions rejected with "debits must equal credits"

Solutions:
- Check for floating-point precision issues
- Verify all amounts are positive
- Ensure currency matches across all postings

### 10.2 Debugging Queries

Check projection status:

```sql
SELECT 
  (SELECT id FROM events ORDER BY created_at DESC LIMIT 1) as latest_event,
  (SELECT last_processed_event_id FROM projector_offsets WHERE projector_name = 'ledger') as processed_event;
```

Check webhook job queue:

```sql
SELECT kind, state, COUNT(*) 
FROM river_job 
WHERE kind = 'webhook_delivery'
GROUP BY kind, state;
```

Check account balance consistency:

```sql
SELECT 
  a.code,
  a.balance as materialized_balance,
  COALESCE(SUM(CASE WHEN p.direction = 'debit' THEN p.amount ELSE -p.amount END), 0) as computed_balance
FROM accounts a
LEFT JOIN postings p ON p.account_id = a.id
WHERE a.ledger_id = 'your-ledger-id'
GROUP BY a.id, a.code, a.balance;
```

## 11. Next Steps and Extensions

### 11.1 Additional Features

- **Multi-currency support**: Handle currency conversion and multi-currency transactions
- **Account hierarchies**: Support parent-child account relationships
- **Transaction reversal**: Implement reversal/void operations
- **Audit logs**: Track all changes to IAM and ledger configuration
- **Reporting**: Build balance sheets, income statements, and trial balances
- **Bulk operations**: Support batch transaction posting
- **Scheduled transactions**: Support recurring transactions

### 11.2 Performance Optimizations

- Implement read-through caching for account balances
- Use materialized views for complex reporting queries
- Partition large tables by time or tenant
- Implement query result caching with Redis

### 11.3 Advanced Multi-Tenancy

- Support dedicated databases for enterprise customers
- Implement tenant-specific rate limits and quotas
- Add tenant-level feature flags
- Support data residency requirements (region-specific databases)

## 12. Summary

You now have a complete implementation of the LaaS platform with:

- Multi-tenant architecture with strong isolation
- CQRS and event sourcing for the ledger core
- Reliable webhook delivery using transactional outbox
- React-based developer dashboard
- Production-ready deployment configuration

The system provides strong consistency guarantees, idempotent operations, and comprehensive observability. Follow the phased development approach to build incrementally and validate each component before moving to the next phase.

For questions or issues, refer to the troubleshooting guide and ensure all tests pass before deploying to production.
