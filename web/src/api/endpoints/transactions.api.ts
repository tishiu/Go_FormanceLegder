import { api } from '../client';

export interface Transaction {
    id: string;
    ledger_id: string;
    reference?: string;
    timestamp: string;
    postings: Posting[];
    metadata?: Record<string, any>;
}

export interface Posting {
    source: string;
    destination: string;
    amount: number;
    asset: string;
}

export interface CreateTransactionInput {
    reference?: string;
    postings: Posting[];
    metadata?: Record<string, any>;
}

export interface TransactionFilters {
    start_date?: string;
    end_date?: string;
    account?: string;
    reference?: string;
    page?: number;
    limit?: number;
}

export const transactionsApi = {
    getByLedger: async (ledgerId: string, filters?: TransactionFilters): Promise<Transaction[]> => {
        const params = new URLSearchParams();
        if (filters?.start_date) params.append('start_date', filters.start_date);
        if (filters?.end_date) params.append('end_date', filters.end_date);
        if (filters?.account) params.append('account', filters.account);
        if (filters?.reference) params.append('reference', filters.reference);
        if (filters?.page) params.append('page', filters.page.toString());
        if (filters?.limit) params.append('limit', filters.limit.toString());

        const response = await api.get<Transaction[]>(
            `/ledgers/${ledgerId}/transactions?${params.toString()}`
        );
        return response.data;
    },

    getById: async (ledgerId: string, transactionId: string): Promise<Transaction> => {
        const response = await api.get<Transaction>(
            `/ledgers/${ledgerId}/transactions/${transactionId}`
        );
        return response.data;
    },

    create: async (ledgerId: string, data: CreateTransactionInput): Promise<Transaction> => {
        const response = await api.post<Transaction>(
            `/ledgers/${ledgerId}/transactions`,
            data
        );
        return response.data;
    },
};

export const queryKeys = {
    transactions: {
        byLedger: (ledgerId: string, filters?: TransactionFilters) =>
            ['transactions', ledgerId, filters] as const,
        detail: (ledgerId: string, transactionId: string) =>
            ['transactions', ledgerId, transactionId] as const,
    },
};
