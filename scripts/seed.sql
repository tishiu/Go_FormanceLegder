-- Create test user
INSERT INTO users (id, email, password_hash)
VALUES ('00000000-0000-0000-0000-000000000001',
        'test@example.com',
        '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy' -- password: "password"
       );

-- Create test organization
INSERT INTO organizations (id, name)
VALUES ('00000000-0000-0000-0000-000000000002', 'Test Org');

-- Link user to organization
INSERT INTO org_users (id, organization_id, user_id, role)
VALUES ('00000000-0000-0000-0000-000000000003',
        '00000000-0000-0000-0000-000000000002',
        '00000000-0000-0000-0000-000000000001',
        'owner');

-- Create test project
INSERT INTO projects (id, organization_id, name, code)
VALUES ('00000000-0000-0000-0000-000000000004',
        '00000000-0000-0000-0000-000000000002',
        'Test Project',
        'test');

-- Create test ledger
INSERT INTO ledgers (id, project_id, name, code, currency)
VALUES ('00000000-0000-0000-0000-000000000005',
        '00000000-0000-0000-0000-000000000004',
        'Sandbox',
        'sandbox',
        'USD');

-- Create test accounts
INSERT INTO accounts (id, ledger_id, code, name, type, balance)
VALUES ('00000000-0000-0000-0000-000000000006',
        '00000000-0000-0000-0000-000000000005',
        'cash',
        'Cash',
        'asset',
        0),
       ('00000000-0000-0000-0000-000000000007',
        '00000000-0000-0000-0000-000000000005',
        'revenue',
        'Revenue',
        'revenue',
        0);

-- Create test API key (raw key: sk_test_12345678901234567890123456789012)
-- Hash computed with HMAC-SHA256 using secret "change-me-in-production"
INSERT INTO api_keys (id, ledger_id, key_hash, prefix, description, is_active)
VALUES ('00000000-0000-0000-0000-000000000008',
        '00000000-0000-0000-0000-000000000005',
        'computed_hash_here', -- Replace with actual hash
        'sk_test_',
        'Test API Key',
        true);