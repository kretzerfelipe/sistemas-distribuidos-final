import { useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeftIcon, PlayIcon, Trash2Icon } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/dialogs'
import { EmptyState, ErrorState, LoadingRows } from '@/components/state'
import { useSession } from '@/core/auth/session'
import { errorMessage } from '@/lib/format'
import { useTeamsById } from '@/modules/teams/use-teams-by-id'
import { matchEventsQuery, matchQuery, useChangeMatchStatus, useDeleteMatch, useRemoveEvent } from './api'
import { EventComposer } from './event-composer'
import { EventTimeline } from './event-timeline'
import { Scoreboard } from './scoreboard'

export function MatchDetailPage({ matchId }: { matchId: string }) {
  const navigate = useNavigate()
  const session = useSession()
  const match = useQuery(matchQuery(matchId))
  const isLive = match.data?.status === 'IN_PROGRESS'
  const events = useQuery({ ...matchEventsQuery(matchId, isLive), enabled: match.isSuccess })
  const { nameOf } = useTeamsById()
  const changeStatus = useChangeMatchStatus(matchId)
  const removeEvent = useRemoveEvent(matchId)
  const removeMatch = useDeleteMatch()
  const [deleteIsOpen, setDeleteIsOpen] = useState(false)
  const isAdmin = session?.user.role === 'ADMIN'

  if (match.isPending) return <LoadingRows rows={2} />
  if (match.isError) return <ErrorState error={match.error} onRetry={() => match.refetch()} />

  const data = match.data
  const backTarget = { championshipId: data.championshipId }

  const advance = (action: 'start' | 'finish') =>
    changeStatus.mutate(action, {
      onSuccess: () => toast.success(action === 'start' ? 'Partida iniciada' : 'Partida encerrada'),
      onError: (error) => toast.error(errorMessage(error)),
    })

  const confirmDelete = () =>
    removeMatch.mutate(matchId, {
      onSuccess: () => {
        toast.success('Partida excluída')
        navigate({ to: '/championships/$championshipId', params: backTarget })
      },
      onError: (error) => {
        setDeleteIsOpen(false)
        toast.error(errorMessage(error))
      },
    })

  return (
    <>
      <Link
        to="/championships/$championshipId"
        params={backTarget}
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeftIcon className="size-4" />
        Voltar ao campeonato
      </Link>

      <Scoreboard match={data} homeName={nameOf(data.homeTeamId)} awayName={nameOf(data.awayTeamId)} />

      {isAdmin ? (
        <div className="mt-4 flex flex-wrap gap-2">
          {data.status === 'SCHEDULED' ? (
            <>
              <Button disabled={changeStatus.isPending} onClick={() => advance('start')}>
                <PlayIcon />
                Iniciar partida
              </Button>
              <Button variant="outline" onClick={() => setDeleteIsOpen(true)}>
                <Trash2Icon />
                Excluir partida
              </Button>
            </>
          ) : null}
          {data.status === 'IN_PROGRESS' ? (
            <Button disabled={changeStatus.isPending} onClick={() => advance('finish')}>
              Encerrar partida
            </Button>
          ) : null}
        </div>
      ) : null}

      {isAdmin && data.status === 'IN_PROGRESS' ? (
        <div className="mt-6">
          <EventComposer match={data} nameOf={nameOf} />
        </div>
      ) : null}

      <section className="mt-8">
        <h2 className="mb-3 text-xl font-semibold">Lances da partida</h2>
        {events.isPending && match.isSuccess ? <LoadingRows rows={2} /> : null}
        {events.isError ? <ErrorState error={events.error} onRetry={() => events.refetch()} /> : null}
        {events.isSuccess && events.data.length === 0 ? (
          <EmptyState
            title="Nenhum lance ainda"
            description={
              data.status === 'SCHEDULED' ? 'Os lances aparecem aqui quando a partida começar.' : 'Nada foi registrado até agora.'
            }
          />
        ) : null}
        {events.isSuccess && events.data.length > 0 ? (
          <EventTimeline
            events={events.data}
            nameOf={nameOf}
            canRemove={isAdmin && data.status === 'IN_PROGRESS'}
            removing={removeEvent.isPending}
            onRemove={(event) =>
              removeEvent.mutate(event.id, {
                onSuccess: () => toast.success('Lance removido'),
                onError: (error) => toast.error(errorMessage(error)),
              })
            }
          />
        ) : null}
        {isLive ? (
          <p className="mt-3 text-xs text-muted-foreground">Atualiza a cada poucos segundos enquanto a partida está ao vivo.</p>
        ) : null}
      </section>

      {isAdmin ? (
        <ConfirmDialog
          open={deleteIsOpen}
          onOpenChange={setDeleteIsOpen}
          title="Excluir esta partida?"
          description="Só partidas agendadas podem ser excluídas. Essa ação não pode ser desfeita."
          confirmLabel="Excluir partida"
          pending={removeMatch.isPending}
          onConfirm={confirmDelete}
        />
      ) : null}
    </>
  )
}
