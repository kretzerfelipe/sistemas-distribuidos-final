import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, ApiError } from '@/lib/api'
import type { Invitation, Player, Team } from '@/types'

export type SaveTeamInput = { name: string; city: string }

export const teamsQuery = queryOptions({
  queryKey: ['teams'],
  queryFn: () => api<Team[]>('/teams'),
})

export const myTeamsQuery = queryOptions({
  queryKey: ['teams', 'mine'],
  queryFn: () => api<Team[]>('/teams/mine'),
})

export const joinedTeamQuery = queryOptions({
  queryKey: ['teams', 'joined'],
  queryFn: async () => {
    try {
      return await api<Team>('/players/me/team')
    } catch (error) {
      const hasNoTeam = error instanceof ApiError && error.status === 404
      if (hasNoTeam) return null
      throw error
    }
  },
})

export const teamQuery = (id: string) =>
  queryOptions({
    queryKey: ['teams', id],
    queryFn: () => api<Team>(`/teams/${id}`),
  })

export const teamPlayersQuery = (id: string) =>
  queryOptions({
    queryKey: ['teams', id, 'players'],
    queryFn: () => api<Player[]>(`/teams/${id}/players`),
  })

export const teamInvitationsQuery = (id: string) =>
  queryOptions({
    queryKey: ['teams', id, 'invitations'],
    queryFn: () => api<Invitation[]>(`/teams/${id}/invitations`),
  })

export function useSaveTeam(teamId?: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: SaveTeamInput) =>
      api<Team>(teamId ? `/teams/${teamId}` : '/teams', { method: teamId ? 'PUT' : 'POST', body: input }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['teams'] }),
  })
}

export function useDeleteTeam() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/teams/${id}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['teams'] }),
  })
}

export function useInvitePlayer(teamId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (playerId: string) =>
      api<Invitation>(`/teams/${teamId}/invitations`, { method: 'POST', body: { playerId } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: teamInvitationsQuery(teamId).queryKey }),
  })
}

export function useCancelInvitation(teamId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (invitationId: string) =>
      api<void>(`/teams/${teamId}/invitations/${invitationId}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: teamInvitationsQuery(teamId).queryKey }),
  })
}

export function useRemovePlayer(teamId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (playerId: string) => api<void>(`/teams/${teamId}/players/${playerId}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: teamPlayersQuery(teamId).queryKey }),
  })
}
