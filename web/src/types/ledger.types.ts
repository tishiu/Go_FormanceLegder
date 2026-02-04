/**
 * Ledger Domain Types
 */

export interface Ledger {
    id: string;
    project_id: string;
    name: string;
    code: string;
    currency: string;
    created_at: string;
}

export interface CreateLedgerInput {
    name: string;
    code: string;
    currency: string;
}

export interface Account {
    id: string;
    ledger_id: string;
    code: string;
    metadata: Record<string, unknown>;
    balance: string; // Using string for precise decimal handling
    created_at: string;
}

export interface CreateAccountInput {
    code: string;
    metadata?: Record<string, unknown>;
}

export interface Transaction {
    id: string;
    ledger_id: string;
    external_id?: string;
    amount: string;
    currency: string;
    occurred_at: string;
    metadata?: Record<string, unknown>;
    created_at: string;
}

export interface Posting {
    id: string;
    transaction_id: string;
    account_id: string;
    amount: string;
    direction: 'debit' | 'credit';
}

export interface CreateTransactionInput {
    external_id?: string;
    amount: string;
    currency: string;
    postings: {
        account_code: string;
        amount: string;
        direction: 'debit' | 'credit';
    }[];
    metadata?: Record<string, unknown>;
}
