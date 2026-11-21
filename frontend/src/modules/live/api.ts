import { queryOptions } from '@tanstack/react-query'
import { notificationApi } from '@/lib/api'
import type { LiveSummary } from '@/types'

const summaryPollingMs = 20_000

export const liveSummaryQuery = (matchId: string, enabled: boolean) =>
  queryOptions({
    queryKey: ['live', matchId, 'summary'],
    queryFn: () => notificationApi<LiveSummary>(`/matches/${matchId}/summary`),
    enabled,
    refetchInterval: summaryPollingMs,
    retry: false,
  })
