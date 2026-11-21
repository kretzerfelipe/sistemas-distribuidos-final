import { useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeftIcon, PencilIcon, Trash2Icon, XIcon } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/dialogs'
import { NativeSelect } from '@/components/native-select'
import { EmptyState, ErrorState, LoadingRows } from '@/components/state'
import { PageHeader } from '@/components/page-header'
import { StatusBadge } from '@/components/status-badge'
import { useSession } from '@/core/auth/session'
import { errorMessage, formatDate, positionLabels } from '@/lib/format'
import { playersQuery } from '@/modules/players/api'
import {
  teamInvitationsQuery,
  teamPlayersQuery,
  teamQuery,
  useCancelInvitation,
  useDeleteTeam,
  useInvitePlayer,
  useRemovePlayer,
} from './api'
import { TeamForm } from './team-form'

function Roster({ teamId, canManage }: { teamId: string; canManage: boolean }) {
  const roster = useQuery(teamPlayersQuery(teamId))
  const removePlayer = useRemovePlayer(teamId)

  return (
    <section className="mb-10">
      <h2 className="mb-3 text-xl font-semibold">Elenco</h2>
      {roster.isPending ? <LoadingRows rows={2} /> : null}
      {roster.isError ? <ErrorState error={roster.error} onRetry={() => roster.refetch()} /> : null}
      {roster.isSuccess && roster.data.length === 0 ? (
        <EmptyState title="Nenhum jogador ainda" description={canManage ? 'Convide jogadores abaixo.' : undefined} />
      ) : null}
      {roster.isSuccess && roster.data.length > 0 ? (
        <ul className="divide-y rounded-xl border bg-card">
          {roster.data.map((player) => (
            <li key={player.id} className="grid grid-cols-[2.5rem_1fr_auto] items-center gap-3 px-4 py-3">
              <span className="font-heading text-xl font-semibold text-primary tabular-nums">{player.shirtNumber}</span>
              <div className="min-w-0">
                <p className="truncate font-medium">{player.name}</p>
                <p className="text-sm text-muted-foreground">{positionLabels[player.position]}</p>
              </div>
              {canManage ? (
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={removePlayer.isPending}
                  onClick={() =>
                    removePlayer.mutate(player.id, {
                      onSuccess: () => toast.success(`${player.name} saiu do time`),
                      onError: (error) => toast.error(errorMessage(error)),
                    })
                  }
                >
                  <XIcon />
                  Remover
                </Button>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  )
}

function Invitations({ teamId }: { teamId: string }) {
  const invitations = useQuery(teamInvitationsQuery(teamId))
  const roster = useQuery(teamPlayersQuery(teamId))
  const players = useQuery(playersQuery)
  const invite = useInvitePlayer(teamId)
  const cancel = useCancelInvitation(teamId)
  const [playerId, setPlayerId] = useState('')

  const rosterIds = new Set((roster.data ?? []).map((player) => player.id))
  const pendingIds = new Set(
    (invitations.data ?? []).filter((invitation) => invitation.status === 'PENDING').map((invitation) => invitation.playerId),
  )
  const invitable = (players.data ?? []).filter((player) => !rosterIds.has(player.id) && !pendingIds.has(player.id))
  const playerName = (id: string) => players.data?.find((player) => player.id === id)?.name ?? 'Jogador desconhecido'

  const sendInvitation = () =>
    invite.mutate(playerId, {
      onSuccess: () => {
        toast.success('Convite enviado')
        setPlayerId('')
      },
      onError: (error) => toast.error(errorMessage(error)),
    })

  return (
    <section>
      <h2 className="mb-3 text-xl font-semibold">Convites</h2>
      <div className="mb-4 flex flex-wrap items-center gap-2 rounded-xl bg-secondary/60 p-3">
        <div className="min-w-48 flex-1">
          <NativeSelect aria-label="Jogador a convidar" value={playerId} onChange={(e) => setPlayerId(e.target.value)}>
            <option value="">Escolha um jogador para convidar</option>
            {invitable.map((player) => (
              <option key={player.id} value={player.id}>
                {player.name} · {positionLabels[player.position]}
              </option>
            ))}
          </NativeSelect>
        </div>
        <Button disabled={playerId === '' || invite.isPending} onClick={sendInvitation}>
          Enviar convite
        </Button>
      </div>

      {invitations.isPending ? <LoadingRows rows={2} /> : null}
      {invitations.isError ? <ErrorState error={invitations.error} /> : null}
      {invitations.isSuccess && invitations.data.length === 0 ? (
        <EmptyState title="Nenhum convite enviado" description="Os jogadores que você convidar aparecem aqui com a resposta." />
      ) : null}
      {invitations.isSuccess && invitations.data.length > 0 ? (
        <ul className="divide-y rounded-xl border bg-card">
          {invitations.data.map((invitation) => (
            <li key={invitation.id} className="flex items-center justify-between gap-3 px-4 py-3">
              <div className="min-w-0">
                <p className="truncate font-medium">{playerName(invitation.playerId)}</p>
                <p className="text-sm text-muted-foreground">Enviado em {formatDate(invitation.createdAt)}</p>
              </div>
              <div className="flex items-center gap-2">
                <StatusBadge status={invitation.status} />
                {invitation.status === 'PENDING' ? (
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={cancel.isPending}
                    onClick={() =>
                      cancel.mutate(invitation.id, {
                        onSuccess: () => toast.success('Convite cancelado'),
                        onError: (error) => toast.error(errorMessage(error)),
                      })
                    }
                  >
                    Cancelar
                  </Button>
                ) : null}
              </div>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  )
}

export function TeamDetailPage({ teamId }: { teamId: string }) {
  const navigate = useNavigate()
  const session = useSession()
  const team = useQuery(teamQuery(teamId))
  const remove = useDeleteTeam()
  const [editIsOpen, setEditIsOpen] = useState(false)
  const [deleteIsOpen, setDeleteIsOpen] = useState(false)

  if (team.isPending) return <LoadingRows rows={2} />
  if (team.isError) return <ErrorState error={team.error} onRetry={() => team.refetch()} />

  const data = team.data
  const isAdmin = session?.user.role === 'ADMIN'
  const isOwner = data.ownerId === session?.user.id
  const canManage = isAdmin || isOwner
  const canInvite = session?.user.role === 'TEAM' && isOwner

  const confirmDelete = () =>
    remove.mutate(teamId, {
      onSuccess: () => {
        toast.success('Time excluído')
        navigate({ to: '/teams' })
      },
      onError: (error) => {
        setDeleteIsOpen(false)
        toast.error(errorMessage(error))
      },
    })

  return (
    <>
      <Link to="/teams" className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeftIcon className="size-4" />
        Todos os times
      </Link>

      <PageHeader
        title={data.name}
        description={data.city}
        actions={
          canManage ? (
            <>
              <Button variant="outline" size="icon" aria-label="Editar time" onClick={() => setEditIsOpen(true)}>
                <PencilIcon />
              </Button>
              <Button variant="outline" size="icon" aria-label="Excluir time" onClick={() => setDeleteIsOpen(true)}>
                <Trash2Icon />
              </Button>
            </>
          ) : null
        }
      />

      <Roster teamId={teamId} canManage={canManage} />
      {canInvite ? <Invitations teamId={teamId} /> : null}

      {canManage ? (
        <>
          <TeamForm key={`${data.name}-${data.city}`} open={editIsOpen} onOpenChange={setEditIsOpen} team={data} />
          <ConfirmDialog
            open={deleteIsOpen}
            onOpenChange={setDeleteIsOpen}
            title="Excluir este time?"
            description="Essa ação não pode ser desfeita. Um time inscrito em um campeonato não pode ser excluído."
            confirmLabel="Excluir time"
            pending={remove.isPending}
            onConfirm={confirmDelete}
          />
        </>
      ) : null}
    </>
  )
}
