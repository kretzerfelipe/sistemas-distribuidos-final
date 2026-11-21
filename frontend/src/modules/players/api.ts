import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, ApiError } from '@/lib/api'
import type { Invitation, Player, Position } from '@/types'

export type SavePlayerInput = { name: string; position: Position; shirtNumber: number }

export const myPlayerQuery = queryOptions({
  queryKey: ['players', 'me'],
  queryFn: async () => {
    try {
      return await api<Player>('/players/me')
    } catch (error) {
      const hasNoProfile = error instanceof ApiError && error.status === 404
      if (hasNoProfile) return null
      throw error
    }
  },
})

export const playersQuery = queryOptions({
  queryKey: ['players'],
  queryFn: () => api<Player[]>('/players'),
})

export const myInvitationsQuery = queryOptions({
  queryKey: ['invitations', 'mine'],
  queryFn: () => api<Invitation[]>('/invitations'),
})

export function useSavePlayer(hasProfile: boolean) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: SavePlayerInput) =>
      api<Player>(hasProfile ? '/players/me' : '/players', { method: hasProfile ? 'PUT' : 'POST', body: input }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: myPlayerQuery.queryKey }),
  })
}

export function useDeleteMyPlayer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api<void>('/players/me', { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: myPlayerQuery.queryKey }),
  })
}

export function useLeaveTeam() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api<void>('/players/me/team', { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['teams'] }),
  })
}

export function useAnswerInvitation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, answer }: { id: string; answer: 'accept' | 'decline' }) =>
      api<Invitation>(`/invitations/${id}/${answer}`, { method: 'POST' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: myInvitationsQuery.queryKey })
      queryClient.invalidateQueries({ queryKey: ['teams'] })
    },
  })
}
