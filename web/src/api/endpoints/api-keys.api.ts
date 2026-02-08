import { api } from '../client';

export interface ApiKey {
    id: string;
    name: string;
    key: string;
    permissions: string[];
    enabled: boolean;
    created_at: string;
    last_used_at?: string;
    expires_at?: string;
}

export interface CreateApiKeyInput {
    name: string;
    permissions: string[];
    expires_at?: string;
}

export const apiKeysApi = {
    getAll: async (): Promise<Omit<ApiKey, 'key'>[]> => {
        const response = await api.get<Omit<ApiKey, 'key'>[]>('/api-keys');
        return response.data;
    },

    getById: async (id: string): Promise<Omit<ApiKey, 'key'>> => {
        const response = await api.get<Omit<ApiKey, 'key'>>(`/api-keys/${id}`);
        return response.data;
    },

    create: async (data: CreateApiKeyInput): Promise<ApiKey> => {
        const response = await api.post<ApiKey>('/api-keys', data);
        return response.data;
    },

    revoke: async (id: string): Promise<void> => {
        await api.delete(`/api-keys/${id}`);
    },

    toggle: async (id: string, enabled: boolean): Promise<Omit<ApiKey, 'key'>> => {
        const response = await api.patch<Omit<ApiKey, 'key'>>(`/api-keys/${id}`, { enabled });
        return response.data;
    },
};

export const queryKeys = {
    apiKeys: {
        all: ['api-keys'] as const,
        detail: (id: string) => ['api-keys', id] as const,
    },
};
