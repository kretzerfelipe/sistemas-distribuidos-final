import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSession } from '@/core/auth/session'
import { joinedTeamQuery, myTeamsQuery } from './api'

export function useMyTeamIds() {
  const session = useSession()
  const role = session?.user.role
  const managesTeams = role === 'TEAM'
  const playsForATeam = role === 'USER'

  const ownedTeams = useQuery({ ...myTeamsQuery, enabled: managesTeams })
  const joinedTeam = useQuery({ ...joinedTeamQuery, enabled: playsForATeam })

  const teamIds = useMemo(() => {
    if (managesTeams) return new Set((ownedTeams.data ?? []).map((team) => team.id))
    if (playsForATeam && joinedTeam.data) return new Set([joinedTeam.data.id])
    return new Set<string>()
  }, [managesTeams, playsForATeam, ownedTeams.data, joinedTeam.data])

  const hasPersonalGames = managesTeams || playsForATeam
  const isLoading = (managesTeams && ownedTeams.isPending) || (playsForATeam && joinedTeam.isPending)

  return { teamIds, hasPersonalGames, isLoading, managesTeams }
}
