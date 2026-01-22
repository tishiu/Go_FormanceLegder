-- Events table (source of truth)
CREATE TABLE IF NOT EXISTS events
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

CREATE INDEX IF NOT EXISTS idx_events_ledger ON events (ledger_id);
CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events (aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events (event_type);
CREATE INDEX IF NOT EXISTS idx_events_created ON events (created_at);

-- Accounts table (read model)
CREATE TABLE IF NOT EXISTS accounts
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

CREATE INDEX IF NOT EXISTS idx_accounts_ledger_code ON accounts (ledger_id, code);

-- Transactions table (read model)
CREATE TABLE IF NOT EXISTS transactions
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

CREATE INDEX IF NOT EXISTS idx_transactions_ledger ON transactions (ledger_id);
CREATE INDEX IF NOT EXISTS idx_transactions_external ON transactions (ledger_id, external_id);

-- Postings table (read model)
CREATE TABLE IF NOT EXISTS postings
(
    id             UUID PRIMARY KEY         DEFAULT gen_random_uuid(),
    ledger_id      UUID            NOT NULL REFERENCES ledgers (id) ON DELETE CASCADE,
    transaction_id UUID            NOT NULL REFERENCES transactions (id) ON DELETE CASCADE,
    account_id     UUID            NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    amount         NUMERIC(38, 10) NOT NULL,
    direction      TEXT            NOT NULL CHECK (direction IN ('debit', 'credit')),
    created_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_postings_ledger ON postings (ledger_id);
CREATE INDEX IF NOT EXISTS idx_postings_transaction ON postings (transaction_id);
CREATE INDEX IF NOT EXISTS idx_postings_account ON postings (account_id);

-- Projector offsets table
CREATE TABLE IF NOT EXISTS projector_offsets
(
    projector_name          TEXT PRIMARY KEY,
    last_processed_event_id UUID NOT NULL
);