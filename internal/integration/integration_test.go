package integration

import (
	"Go_FormanceLegder/internal/ledger"
	"Go_FormanceLegder/internal/webhook"
	"context"
	"testing"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPostTransactionEndToEnd(t *testing.T) {
	ctx := context.Background()

	// Setup test container
	container, dbURL, err := setupPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("failed to setup postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	// Setup database
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	// Run migrations
	runMigrations(t, pool)

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

func setupPostgresContainer(ctx context.Context) (testcontainers.Container, string, error) {
	// Create PostgreSQL container
	container, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16"),
		postgres.WithDatabase("ledger_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, "", err
	}

	// Get connection string
	dbURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, "", err
	}

	return container, dbURL, nil
}

func runMigrations(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()

	// Run SQL migrations
	migrations := []string{
		migrations001CreateIAMTables,
		migrations002CreateLedgerTables,
		migrations003CreateWebhookTables,
	}

	for _, migration := range migrations {
		_, err := pool.Exec(ctx, migration)
		if err != nil {
			t.Fatalf("failed to run migration: %v", err)
		}
	}

	// Run River migrations
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}
	
	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		t.Fatalf("failed to run river migrations: %v", err)
	}
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

	// Create organization
	_, err := pool.Exec(ctx, `
		INSERT INTO organizations (id, name)
		VALUES ('00000000-0000-0000-0000-000000000002', 'Test Org')
	`)
	if err != nil {
		t.Fatalf("failed to seed organization: %v", err)
	}

	// Create project
	_, err = pool.Exec(ctx, `
		INSERT INTO projects (id, organization_id, name, code)
		VALUES ('00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000002', 'Test Project', 'test')
	`)
	if err != nil {
		t.Fatalf("failed to seed project: %v", err)
	}

	// Create ledger
	_, err = pool.Exec(ctx, `
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

// Migration constants
const migrations001CreateIAMTables = `
-- Users table
CREATE TABLE users
(
    id            UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Organizations table
CREATE TABLE organizations
(
    id         UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Organization users junction table
CREATE TABLE org_users
(
    id              UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    organization_id UUID        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role            TEXT        NOT NULL CHECK (role IN ('owner', 'developer')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, user_id)
);

CREATE INDEX idx_org_users_org ON org_users (organization_id);
CREATE INDEX idx_org_users_user ON org_users (user_id);

-- Projects table
CREATE TABLE projects
(
    id              UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    organization_id UUID        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name            TEXT        NOT NULL,
    code            TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, code)
);

CREATE INDEX idx_projects_org ON projects (organization_id);

-- Ledgers table
CREATE TABLE ledgers
(
    id         UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    project_id UUID        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    code       TEXT        NOT NULL,
    currency   TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, code)
);

CREATE INDEX idx_ledgers_project ON ledgers (project_id);

-- API Keys table
CREATE TABLE api_keys
(
    id          UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    ledger_id   UUID        NOT NULL REFERENCES ledgers (id) ON DELETE CASCADE,
    key_hash    TEXT        NOT NULL UNIQUE,
    prefix      TEXT        NOT NULL,
    description TEXT,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_ledger ON api_keys (ledger_id);
CREATE INDEX idx_api_keys_hash ON api_keys (key_hash);
`

const migrations002CreateLedgerTables = `
-- Events table (source of truth)
CREATE TABLE events
(
    id              UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    ledger_id       UUID        NOT NULL REFERENCES ledgers (id) ON DELETE CASCADE,
    aggregate_type  TEXT        NOT NULL,
    aggregate_id    UUID        NOT NULL,
    event_type      TEXT        NOT NULL,
    payload         JSONB       NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    idempotency_key TEXT,
    UNIQUE (ledger_id, idempotency_key)
);

CREATE INDEX idx_events_ledger ON events (ledger_id);
CREATE INDEX idx_events_aggregate ON events (aggregate_type, aggregate_id);
CREATE INDEX idx_events_type ON events (event_type);
CREATE INDEX idx_events_created ON events (created_at);

-- Accounts table (read model)
CREATE TABLE accounts
(
    id         UUID PRIMARY KEY         DEFAULT gen_random_uuid(),
    ledger_id  UUID            NOT NULL REFERENCES ledgers (id) ON DELETE CASCADE,
    code       TEXT            NOT NULL,
    name       TEXT            NOT NULL,
    type       TEXT            NOT NULL CHECK (type IN ('asset', 'liability', 'equity', 'revenue', 'expense')),
    balance    NUMERIC(38, 10) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (ledger_id, code)
);

CREATE INDEX idx_accounts_ledger_code ON accounts (ledger_id, code);

-- Transactions table (read model)
CREATE TABLE transactions
(
    id          UUID PRIMARY KEY,
    ledger_id   UUID            NOT NULL REFERENCES ledgers (id) ON DELETE CASCADE,
    external_id TEXT,
    amount      NUMERIC(38, 10) NOT NULL,
    currency    TEXT            NOT NULL,
    occurred_at TIMESTAMPTZ     NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (id, ledger_id)
);

CREATE INDEX idx_transactions_ledger ON transactions (ledger_id);
CREATE INDEX idx_transactions_external ON transactions (ledger_id, external_id);

-- Postings table (read model)
CREATE TABLE postings
(
    id             UUID PRIMARY KEY         DEFAULT gen_random_uuid(),
    ledger_id      UUID            NOT NULL REFERENCES ledgers (id) ON DELETE CASCADE,
    transaction_id UUID            NOT NULL REFERENCES transactions (id) ON DELETE CASCADE,
    account_id     UUID            NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    amount         NUMERIC(38, 10) NOT NULL,
    direction      TEXT            NOT NULL CHECK (direction IN ('debit', 'credit')),
    created_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_postings_ledger ON postings (ledger_id);
CREATE INDEX idx_postings_transaction ON postings (transaction_id);
CREATE INDEX idx_postings_account ON postings (account_id);

-- Projector offsets table
CREATE TABLE projector_offsets
(
    projector_name          TEXT PRIMARY KEY,
    last_processed_event_id UUID NOT NULL
);
`

const migrations003CreateWebhookTables = `
-- Webhook endpoints table
CREATE TABLE webhook_endpoints
(
    id         UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    ledger_id  UUID        NOT NULL REFERENCES ledgers (id) ON DELETE CASCADE,
    url        TEXT        NOT NULL,
    secret     TEXT        NOT NULL,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_endpoints_ledger ON webhook_endpoints (ledger_id);

-- Webhook deliveries table
CREATE TABLE webhook_deliveries
(
    id                  UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    event_id            UUID        NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    webhook_endpoint_id UUID        NOT NULL REFERENCES webhook_endpoints (id) ON DELETE CASCADE,
    status              TEXT        NOT NULL CHECK (status IN ('success', 'retryable_error', 'non_retryable_error')),
    attempt             INT         NOT NULL,
    last_attempt_at     TIMESTAMPTZ,
    http_status         INT,
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_event ON webhook_deliveries (event_id);
CREATE INDEX idx_webhook_deliveries_endpoint ON webhook_deliveries (webhook_endpoint_id);
`
