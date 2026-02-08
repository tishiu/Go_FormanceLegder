import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ledgersApi, ledgerQueryKeys } from '../api/endpoints';
import type { CreateLedgerInput } from '../types';

export function useLedgers() {
    return useQuery({
        queryKey: ledgerQueryKeys.ledgers.all,
        queryFn: () => ledgersApi.getAll(),
    });
}

export function useLedger(id: string) {
    return useQuery({
        queryKey: ledgerQueryKeys.ledgers.detail(id),
        queryFn: () => ledgersApi.getById(id),
        enabled: !!id,
    });
}

export function useCreateLedger() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateLedgerInput) => ledgersApi.create(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ledgerQueryKeys.ledgers.all });
        },
    });
}

export function useDeleteLedger() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => ledgersApi.delete(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ledgerQueryKeys.ledgers.all });
        },
    });
}
