import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { notificationApi } from '@/lib/api'
import type { AppNotification } from '@/types'

const pollingMs = 15_000

export const notificationsQuery = (enabled = true) =>
  queryOptions({
    queryKey: ['notifications'],
    enabled,
    queryFn: () => notificationApi<AppNotification[]>('/notifications'),
    refetchInterval: pollingMs,
  })

export function useMarkAsRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => notificationApi<void>(`/notifications/${id}/read`, { method: 'POST' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  })
}
