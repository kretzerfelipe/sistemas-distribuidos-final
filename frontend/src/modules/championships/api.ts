import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { Championship, Team } from '@/types'

export type SaveChampionshipInput = { name: string; slotsTotal: number }

export const championshipsQuery = queryOptions({
  queryKey: ['championships'],
  queryFn: () => api<Championship[]>('/championships'),
})

export const championshipQuery = (id: string) =>
  queryOptions({
    queryKey: ['championships', id],
    queryFn: () => api<Championship>(`/championships/${id}`),
  })

export const enrolledTeamsQuery = (id: string) =>
  queryOptions({
    queryKey: ['championships', id, 'teams'],
    queryFn: () => api<Team[]>(`/championships/${id}/teams`),
  })

export function useSaveChampionship(id?: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: SaveChampionshipInput) =>
      api<Championship>(id ? `/championships/${id}` : '/championships', {
        method: id ? 'PUT' : 'POST',
        body: input,
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: championshipsQuery.queryKey }),
  })
}

export function useDeleteChampionship() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/championships/${id}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: championshipsQuery.queryKey }),
  })
}

export function useChangeChampionshipStatus(id: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (action: 'start' | 'finish') => api<Championship>(`/championships/${id}/${action}`, { method: 'POST' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: championshipsQuery.queryKey }),
  })
}

export function useEnrollTeam(id: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (teamId: string) => api<Championship>(`/championships/${id}/teams`, { method: 'POST', body: { teamId } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: championshipsQuery.queryKey }),
  })
}

export function useUnenrollTeam(id: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (teamId: string) => api<void>(`/championships/${id}/teams/${teamId}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: championshipsQuery.queryKey }),
  })
}
