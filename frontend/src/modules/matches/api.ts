import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { Match, MatchEvent, MatchEventType } from '@/types'

export type CreateMatchInput = {
  championshipId: string
  homeTeamId: string
  awayTeamId: string
  scheduledAt: string
}

export type RegisterEventInput = {
  type: MatchEventType
  minute: number
  teamId: string
  playerId: string | null
  secondaryPlayerId: string | null
}

const livePollingMs = 4000

export const championshipMatchesQuery = (championshipId: string) =>
  queryOptions({
    queryKey: ['matches', 'championship', championshipId],
    queryFn: () => api<Match[]>(`/championships/${championshipId}/matches`),
  })

export const matchQuery = (id: string) =>
  queryOptions({
    queryKey: ['matches', id],
    queryFn: () => api<Match>(`/matches/${id}`),
    refetchInterval: (query) => (query.state.data?.status === 'IN_PROGRESS' ? livePollingMs : false),
  })

export const matchEventsQuery = (id: string, isLive: boolean) =>
  queryOptions({
    queryKey: ['matches', id, 'events'],
    queryFn: () => api<MatchEvent[]>(`/matches/${id}/events`),
    refetchInterval: isLive ? livePollingMs : false,
  })

export function useCreateMatch() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateMatchInput) => api<Match>('/matches', { method: 'POST', body: input }),
    onSuccess: (_match, input) =>
      queryClient.invalidateQueries({ queryKey: championshipMatchesQuery(input.championshipId).queryKey }),
  })
}

export function useDeleteMatch() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/matches/${id}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['matches'] }),
  })
}

export function useChangeMatchStatus(id: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (action: 'start' | 'finish') => api<Match>(`/matches/${id}/${action}`, { method: 'POST' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['matches'] }),
  })
}

export function useRegisterEvent(matchId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: RegisterEventInput) =>
      api<MatchEvent>(`/matches/${matchId}/events`, { method: 'POST', body: input }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['matches'] }),
  })
}

export function useRemoveEvent(matchId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (eventId: string) => api<void>(`/matches/${matchId}/events/${eventId}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['matches'] }),
  })
}
