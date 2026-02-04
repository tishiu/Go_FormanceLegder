-- Users table
CREATE TABLE IF NOT EXISTS users
(
    id            UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Organizations table
CREATE TABLE IF NOT EXISTS organizations
(
    id         UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Organization users junction table
CREATE TABLE IF NOT EXISTS org_users
(
    id              UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    organization_id UUID        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role            TEXT        NOT NULL CHECK (role IN ('owner', 'developer')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_org_users_org ON org_users (organization_id);
CREATE INDEX IF NOT EXISTS idx_org_users_user ON org_users (user_id);

-- Projects table
CREATE TABLE IF NOT EXISTS projects
(
    id              UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    organization_id UUID        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name            TEXT        NOT NULL,
    code            TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, code)
);

CREATE INDEX IF NOT EXISTS idx_projects_org ON projects (organization_id);

-- Ledgers table
CREATE TABLE IF NOT EXISTS ledgers
(
    id         UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    project_id UUID        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    code       TEXT        NOT NULL,
    currency   TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, code)
);

CREATE INDEX IF NOT EXISTS idx_ledgers_project ON ledgers (project_id);

-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys
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

CREATE INDEX IF NOT EXISTS idx_api_keys_ledger ON api_keys (ledger_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys (key_hash);