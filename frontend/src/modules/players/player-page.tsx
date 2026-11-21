import { useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ConfirmDialog } from '@/components/dialogs'
import { NativeSelect } from '@/components/native-select'
import { ErrorState, LoadingRows } from '@/components/state'
import { PageHeader } from '@/components/page-header'
import { errorMessage, positionLabels, positions } from '@/lib/format'
import type { Player, Position } from '@/types'
import { myPlayerQuery, useDeleteMyPlayer, useLeaveTeam, useSavePlayer } from './api'

const defaultShirtNumber = 10

function PlayerForm({ player }: { player: Player | null }) {
  const hasProfile = player !== null
  const save = useSavePlayer(hasProfile)
  const [name, setName] = useState(player?.name ?? '')
  const [position, setPosition] = useState<Position>(player?.position ?? 'MIDFIELDER')
  const [shirtNumber, setShirtNumber] = useState(player?.shirtNumber ?? defaultShirtNumber)

  const submit = (event: FormEvent) => {
    event.preventDefault()
    save.mutate(
      { name, position, shirtNumber },
      { onSuccess: () => toast.success(hasProfile ? 'Perfil atualizado' : 'Perfil de jogador criado') },
    )
  }

  return (
    <form onSubmit={submit} className="max-w-md space-y-4 rounded-xl border bg-card p-5">
      <div className="space-y-1.5">
        <Label htmlFor="player-name">Nome na súmula</Label>
        <Input id="player-name" required value={name} onChange={(e) => setName(e.target.value)} />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-1.5">
          <Label htmlFor="player-position">Posição</Label>
          <NativeSelect id="player-position" value={position} onChange={(e) => setPosition(e.target.value as Position)}>
            {positions.map((value) => (
              <option key={value} value={value}>
                {positionLabels[value]}
              </option>
            ))}
          </NativeSelect>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="player-number">Número da camisa</Label>
          <Input
            id="player-number"
            type="number"
            min={1}
            max={99}
            required
            value={shirtNumber}
            onChange={(e) => setShirtNumber(Number(e.target.value))}
          />
        </div>
      </div>
      {save.isError ? (
        <p role="alert" className="text-sm text-destructive">
          {errorMessage(save.error)}
        </p>
      ) : null}
      <Button type="submit" disabled={save.isPending}>
        {hasProfile ? 'Salvar alterações' : 'Criar perfil de jogador'}
      </Button>
    </form>
  )
}

export function PlayerPage() {
  const player = useQuery(myPlayerQuery)
  const deleteProfile = useDeleteMyPlayer()
  const leaveTeam = useLeaveTeam()
  const [deleteIsOpen, setDeleteIsOpen] = useState(false)

  if (player.isPending) return <LoadingRows rows={2} />
  if (player.isError) return <ErrorState error={player.error} onRetry={() => player.refetch()} />

  const profile = player.data

  return (
    <>
      <PageHeader
        title="Meu jogador"
        description={
          profile
            ? 'É assim que os gestores de time veem você ao convidar jogadores.'
            : 'Crie um perfil de jogador para os gestores de time poderem convidar você.'
        }
      />

      <PlayerForm key={profile?.id ?? 'new'} player={profile} />

      {profile ? (
        <div className="mt-8 flex flex-wrap gap-2">
          <Button
            variant="outline"
            disabled={leaveTeam.isPending}
            onClick={() =>
              leaveTeam.mutate(undefined, {
                onSuccess: () => toast.success('Você saiu do time'),
                onError: (error) => toast.error(errorMessage(error)),
              })
            }
          >
            Sair do meu time
          </Button>
          <Button variant="outline" onClick={() => setDeleteIsOpen(true)}>
            Excluir perfil
          </Button>
        </div>
      ) : null}

      <ConfirmDialog
        open={deleteIsOpen}
        onOpenChange={setDeleteIsOpen}
        title="Excluir seu perfil de jogador?"
        description="Você sai do seu time e perde os convites pendentes."
        confirmLabel="Excluir perfil"
        pending={deleteProfile.isPending}
        onConfirm={() =>
          deleteProfile.mutate(undefined, {
            onSuccess: () => {
              setDeleteIsOpen(false)
              toast.success('Perfil excluído')
            },
            onError: (error) => {
              setDeleteIsOpen(false)
              toast.error(errorMessage(error))
            },
          })
        }
      />
    </>
  )
}
