/**
 * API and Integration Types
 */

export interface ApiKey {
    id: string;
    ledger_id: string;
    name: string;
    prefix: string; // First 8 chars shown
    created_at: string;
    last_used_at?: string;
}

export interface CreateApiKeyInput {
    name: string;
}

export interface CreateApiKeyResponse {
    api_key: ApiKey;
    secret: string; // Full key, only shown once
}

export interface Webhook {
    id: string;
    ledger_id: string;
    url: string;
    events: WebhookEvent[];
    secret?: string;
    is_active: boolean;
    created_at: string;
    last_triggered_at?: string;
}

export type WebhookEvent =
    | 'transaction.created'
    | 'account.created'
    | 'account.updated';

export interface CreateWebhookInput {
    url: string;
    events: WebhookEvent[];
}

/**
 * API Response Types
 */
export interface ApiResponse<T> {
    data: T;
    message?: string;
}

export interface PaginatedResponse<T> {
    data: T[];
    total: number;
    page: number;
    limit: number;
    has_more: boolean;
}

export interface ApiError {
    message: string;
    code?: string;
    details?: Record<string, string[]>;
}
