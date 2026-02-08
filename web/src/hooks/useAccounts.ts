import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { accountsApi, accountQueryKeys } from '../api/endpoints';
import type { CreateAccountInput } from '../types';

export function useAccounts(ledgerId: string) {
    return useQuery({
        queryKey: accountQueryKeys.accounts.byLedger(ledgerId),
        queryFn: () => accountsApi.getByLedger(ledgerId),
        enabled: !!ledgerId,
    });
}

export function useAccount(ledgerId: string, accountId: string) {
    return useQuery({
        queryKey: accountQueryKeys.accounts.detail(ledgerId, accountId),
        queryFn: () => accountsApi.getById(ledgerId, accountId),
        enabled: !!ledgerId && !!accountId,
    });
}

export function useCreateAccount(ledgerId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateAccountInput) => accountsApi.create(ledgerId, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: accountQueryKeys.accounts.byLedger(ledgerId) });
        },
    });
}
