import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { transactionsApi, queryKeys, TransactionFilters } from '../api/endpoints/transactions.api';
import type { CreateTransactionInput } from '../api/endpoints/transactions.api';

export function useTransactions(ledgerId: string, filters?: TransactionFilters) {
    return useQuery({
        queryKey: queryKeys.transactions.byLedger(ledgerId, filters),
        queryFn: () => transactionsApi.getByLedger(ledgerId, filters),
        enabled: !!ledgerId,
    });
}

export function useTransaction(ledgerId: string, transactionId: string) {
    return useQuery({
        queryKey: queryKeys.transactions.detail(ledgerId, transactionId),
        queryFn: () => transactionsApi.getById(ledgerId, transactionId),
        enabled: !!ledgerId && !!transactionId,
    });
}

export function useCreateTransaction(ledgerId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateTransactionInput) => transactionsApi.create(ledgerId, data),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: ['transactions', ledgerId],
            });
        },
    });
}
