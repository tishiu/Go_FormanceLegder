import { api } from '../client';

export interface Webhook {
    id: string;
    url: string;
    events: string[];
    enabled: boolean;
    created_at: string;
    updated_at: string;
    metadata?: Record<string, any>;
}

export interface WebhookLog {
    id: string;
    webhook_id: string;
    event_type: string;
    payload: Record<string, any>;
    status_code: number;
    response: string;
    created_at: string;
}

export interface CreateWebhookInput {
    url: string;
    events: string[];
    enabled?: boolean;
    metadata?: Record<string, any>;
}

export const webhooksApi = {
    getAll: async (): Promise<Webhook[]> => {
        const response = await api.get<Webhook[]>('/webhooks');
        return response.data;
    },

    getById: async (id: string): Promise<Webhook> => {
        const response = await api.get<Webhook>(`/webhooks/${id}`);
        return response.data;
    },

    create: async (data: CreateWebhookInput): Promise<Webhook> => {
        const response = await api.post<Webhook>('/webhooks', data);
        return response.data;
    },

    update: async (id: string, data: Partial<CreateWebhookInput>): Promise<Webhook> => {
        const response = await api.put<Webhook>(`/webhooks/${id}`, data);
        return response.data;
    },

    delete: async (id: string): Promise<void> => {
        await api.delete(`/webhooks/${id}`);
    },

    test: async (id: string): Promise<{ success: boolean; message: string }> => {
        const response = await api.post<{ success: boolean; message: string }>(
            `/webhooks/${id}/test`
        );
        return response.data;
    },

    getLogs: async (id: string): Promise<WebhookLog[]> => {
        const response = await api.get<WebhookLog[]>(`/webhooks/${id}/logs`);
        return response.data;
    },
};

export const queryKeys = {
    webhooks: {
        all: ['webhooks'] as const,
        detail: (id: string) => ['webhooks', id] as const,
        logs: (id: string) => ['webhooks', id, 'logs'] as const,
    },
};
