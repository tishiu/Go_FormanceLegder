import { api } from '../client';
import type { Account, CreateAccountInput } from '../../types';

export const accountsApi = {
    getByLedger: async (ledgerId: string): Promise<Account[]> => {
        const response = await api.get<Account[]>(`/ledgers/${ledgerId}/accounts`);
        return response.data;
    },

    getById: async (ledgerId: string, accountId: string): Promise<Account> => {
        const response = await api.get<Account>(`/ledgers/${ledgerId}/accounts/${accountId}`);
        return response.data;
    },

    create: async (ledgerId: string, data: CreateAccountInput): Promise<Account> => {
        const response = await api.post<Account>(`/ledgers/${ledgerId}/accounts`, data);
        return response.data;
    },
};

export const queryKeys = {
    accounts: {
        byLedger: (ledgerId: string) => ['accounts', ledgerId] as const,
        detail: (ledgerId: string, accountId: string) => ['accounts', ledgerId, accountId] as const,
    },
};
