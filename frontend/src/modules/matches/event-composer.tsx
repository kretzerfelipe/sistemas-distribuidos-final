import { useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect } from '@/components/native-select'
import { errorMessage, eventTypeLabels, eventTypes } from '@/lib/format'
import { teamPlayersQuery } from '@/modules/teams/api'
import type { Match, MatchEventType } from '@/types'
import { useRegisterEvent } from './api'

type EventComposerProps = {
  match: Match
  nameOf: (teamId: string) => string
}

export function EventComposer({ match, nameOf }: EventComposerProps) {
  const register = useRegisterEvent(match.id)
  const [type, setType] = useState<MatchEventType>('GOAL')
  const [minute, setMinute] = useState(1)
  const [teamId, setTeamId] = useState(match.homeTeamId)
  const [playerId, setPlayerId] = useState('')
  const [secondaryPlayerId, setSecondaryPlayerId] = useState('')
  const players = useQuery(teamPlayersQuery(teamId))
  const isSubstitution = type === 'SUBSTITUTION'
  const roster = players.data ?? []

  const changeTeam = (nextTeamId: string) => {
    setTeamId(nextTeamId)
    setPlayerId('')
    setSecondaryPlayerId('')
  }

  const submit = (event: FormEvent) => {
    event.preventDefault()
    register.mutate(
      {
        type,
        minute,
        teamId,
        playerId: playerId || null,
        secondaryPlayerId: isSubstitution ? secondaryPlayerId || null : null,
      },
      {
        onSuccess: () => {
          toast.success(`${eventTypeLabels[type]} registrado`)
          setPlayerId('')
          setSecondaryPlayerId('')
        },
        onError: (error) => toast.error(errorMessage(error)),
      },
    )
  }

  return (
    <form onSubmit={submit} className="rounded-xl border bg-card p-4 sm:p-5">
      <h2 className="mb-4 text-lg font-semibold">Registrar lance</h2>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label htmlFor="event-type">Lance</Label>
          <NativeSelect id="event-type" value={type} onChange={(e) => setType(e.target.value as MatchEventType)}>
            {eventTypes.map((value) => (
              <option key={value} value={value}>
                {eventTypeLabels[value]}
              </option>
            ))}
          </NativeSelect>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="event-minute">Minuto</Label>
          <Input
            id="event-minute"
            type="number"
            min={0}
            max={130}
            required
            value={minute}
            onChange={(e) => setMinute(Number(e.target.value))}
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="event-team">Time</Label>
          <NativeSelect id="event-team" value={teamId} onChange={(e) => changeTeam(e.target.value)}>
            <option value={match.homeTeamId}>{nameOf(match.homeTeamId)}</option>
            <option value={match.awayTeamId}>{nameOf(match.awayTeamId)}</option>
          </NativeSelect>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="event-player">{isSubstitution ? 'Jogador que entra' : 'Jogador (opcional)'}</Label>
          <NativeSelect
            id="event-player"
            required={isSubstitution}
            value={playerId}
            onChange={(e) => setPlayerId(e.target.value)}
          >
            <option value="">{isSubstitution ? 'Escolha um jogador' : 'Sem jogador'}</option>
            {roster.map((player) => (
              <option key={player.id} value={player.id}>
                {player.shirtNumber}. {player.name}
              </option>
            ))}
          </NativeSelect>
        </div>
        {isSubstitution ? (
          <div className="space-y-1.5 sm:col-span-2">
            <Label htmlFor="event-secondary">Jogador que sai</Label>
            <NativeSelect
              id="event-secondary"
              required
              value={secondaryPlayerId}
              onChange={(e) => setSecondaryPlayerId(e.target.value)}
            >
              <option value="">Escolha um jogador</option>
              {roster
                .filter((player) => player.id !== playerId)
                .map((player) => (
                  <option key={player.id} value={player.id}>
                    {player.shirtNumber}. {player.name}
                  </option>
                ))}
            </NativeSelect>
          </div>
        ) : null}
      </div>
      {roster.length === 0 && players.isSuccess ? (
        <p className="mt-3 text-sm text-muted-foreground">Esse time ainda não tem jogadores, então o lance será salvo sem jogador.</p>
      ) : null}
      <div className="mt-5 flex justify-end">
        <Button type="submit" disabled={register.isPending}>
          Registrar lance
        </Button>
      </div>
    </form>
  )
}
