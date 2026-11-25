import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, ApiError } from '@/lib/api'
import type { Subscription } from '@/types'

const pendingPollingMs = 2000

export const mySubscriptionQuery = (enabled = true) =>
  queryOptions({
    queryKey: ['subscriptions', 'mine'],
    enabled,
    queryFn: async () => {
      try {
        return await api<Subscription>('/subscriptions/mine')
      } catch (error) {
        const hasNoSubscription = error instanceof ApiError && error.status === 404
        if (hasNoSubscription) return null
        throw error
      }
    },
    refetchInterval: (query) => (query.state.data?.status === 'PENDING' ? pendingPollingMs : false),
  })

export function useSubscribe() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api<Subscription>('/subscriptions', { method: 'POST' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['subscriptions'] }),
  })
}
