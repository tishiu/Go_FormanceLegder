import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { webhooksApi, queryKeys } from '../api/endpoints/webhooks.api';
import type { CreateWebhookInput } from '../api/endpoints/webhooks.api';

export function useWebhooks() {
    return useQuery({
        queryKey: queryKeys.webhooks.all,
        queryFn: () => webhooksApi.getAll(),
    });
}

export function useWebhook(id: string) {
    return useQuery({
        queryKey: queryKeys.webhooks.detail(id),
        queryFn: () => webhooksApi.getById(id),
        enabled: !!id,
    });
}

export function useWebhookLogs(id: string) {
    return useQuery({
        queryKey: queryKeys.webhooks.logs(id),
        queryFn: () => webhooksApi.getLogs(id),
        enabled: !!id,
    });
}

export function useCreateWebhook() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateWebhookInput) => webhooksApi.create(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: queryKeys.webhooks.all });
        },
    });
}

export function useUpdateWebhook() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: Partial<CreateWebhookInput> }) =>
            webhooksApi.update(id, data),
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({ queryKey: queryKeys.webhooks.all });
            queryClient.invalidateQueries({ queryKey: queryKeys.webhooks.detail(variables.id) });
        },
    });
}

export function useDeleteWebhook() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => webhooksApi.delete(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: queryKeys.webhooks.all });
        },
    });
}

export function useTestWebhook() {
    return useMutation({
        mutationFn: (id: string) => webhooksApi.test(id),
    });
}
