import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { teamsQuery } from './api'

export function useTeamsById() {
  const teams = useQuery(teamsQuery)
  const byId = useMemo(() => new Map((teams.data ?? []).map((team) => [team.id, team])), [teams.data])
  const nameOf = (id: string) => byId.get(id)?.name ?? 'Unknown team'
  return { teams, byId, nameOf }
}
