import { useState, type FormEvent } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { FormDialog } from '@/components/dialogs'
import { NativeSelect } from '@/components/native-select'
import { errorMessage } from '@/lib/format'
import type { Team } from '@/types'
import { useCreateMatch } from './api'

type MatchFormProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  championshipId: string
  teams: Team[]
}

export function MatchForm({ open, onOpenChange, championshipId, teams }: MatchFormProps) {
  const create = useCreateMatch()
  const [homeTeamId, setHomeTeamId] = useState('')
  const [awayTeamId, setAwayTeamId] = useState('')
  const [scheduledAt, setScheduledAt] = useState('')
  const sameTeamPicked = homeTeamId !== '' && homeTeamId === awayTeamId

  const submit = (event: FormEvent) => {
    event.preventDefault()
    create.mutate(
      { championshipId, homeTeamId, awayTeamId, scheduledAt: new Date(scheduledAt).toISOString() },
      {
        onSuccess: () => {
          toast.success('Partida agendada')
          onOpenChange(false)
          setHomeTeamId('')
          setAwayTeamId('')
          setScheduledAt('')
        },
      },
    )
  }

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Agendar partida"
      description="Os dois times precisam estar inscritos neste campeonato."
    >
      <form onSubmit={submit} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="home-team">Time da casa</Label>
          <NativeSelect id="home-team" required value={homeTeamId} onChange={(e) => setHomeTeamId(e.target.value)}>
            <option value="">Escolha um time</option>
            {teams.map((team) => (
              <option key={team.id} value={team.id}>
                {team.name}
              </option>
            ))}
          </NativeSelect>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="away-team">Time visitante</Label>
          <NativeSelect id="away-team" required value={awayTeamId} onChange={(e) => setAwayTeamId(e.target.value)}>
            <option value="">Escolha um time</option>
            {teams.map((team) => (
              <option key={team.id} value={team.id}>
                {team.name}
              </option>
            ))}
          </NativeSelect>
          {sameTeamPicked ? <p className="text-xs text-destructive">Escolha dois times diferentes.</p> : null}
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="scheduled-at">Início</Label>
          <Input
            id="scheduled-at"
            type="datetime-local"
            required
            value={scheduledAt}
            onChange={(e) => setScheduledAt(e.target.value)}
          />
        </div>
        {create.isError ? (
          <p role="alert" className="text-sm text-destructive">
            {errorMessage(create.error)}
          </p>
        ) : null}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancelar
          </Button>
          <Button type="submit" disabled={create.isPending || sameTeamPicked}>
            Agendar partida
          </Button>
        </div>
      </form>
    </FormDialog>
  )
}
