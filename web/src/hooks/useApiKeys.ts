import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiKeysApi, queryKeys } from '../api/endpoints/api-keys.api';
import type { CreateApiKeyInput } from '../api/endpoints/api-keys.api';

export function useApiKeys() {
    return useQuery({
        queryKey: queryKeys.apiKeys.all,
        queryFn: () => apiKeysApi.getAll(),
    });
}

export function useApiKey(id: string) {
    return useQuery({
        queryKey: queryKeys.apiKeys.detail(id),
        queryFn: () => apiKeysApi.getById(id),
        enabled: !!id,
    });
}

export function useCreateApiKey() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateApiKeyInput) => apiKeysApi.create(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.all });
        },
    });
}

export function useRevokeApiKey() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => apiKeysApi.revoke(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.all });
        },
    });
}

export function useToggleApiKey() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
            apiKeysApi.toggle(id, enabled),
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.all });
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.detail(variables.id) });
        },
    });
}
