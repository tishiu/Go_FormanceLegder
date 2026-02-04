import { api } from '../client';
import type { Ledger, CreateLedgerInput } from '../../types';

export const ledgersApi = {
    getAll: async (): Promise<Ledger[]> => {
        const response = await api.get<Ledger[]>('/ledgers');
        return response.data;
    },

    getById: async (id: string): Promise<Ledger> => {
        const response = await api.get<Ledger>(`/ledgers/${id}`);
        return response.data;
    },

    create: async (data: CreateLedgerInput): Promise<Ledger> => {
        const response = await api.post<Ledger>('/ledgers', data);
        return response.data;
    },

    delete: async (id: string): Promise<void> => {
        await api.delete(`/ledgers/${id}`);
    },
};

export const queryKeys = {
    ledgers: {
        all: ['ledgers'] as const,
        detail: (id: string) => ['ledgers', id] as const,
    },
};
