import { useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeftIcon, PencilIcon, PlayIcon, PlusIcon, Trash2Icon, XIcon } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/dialogs'
import { NativeSelect } from '@/components/native-select'
import { EmptyState, ErrorState, LoadingRows } from '@/components/state'
import { PageHeader } from '@/components/page-header'
import { StatusBadge } from '@/components/status-badge'
import { useSession } from '@/core/auth/session'
import { errorMessage } from '@/lib/format'
import { championshipMatchesQuery } from '@/modules/matches/api'
import { MatchForm } from '@/modules/matches/match-form'
import { MatchRow } from '@/modules/matches/match-row'
import { myTeamsQuery, teamsQuery } from '@/modules/teams/api'
import { useTeamsById } from '@/modules/teams/use-teams-by-id'
import {
  championshipQuery,
  enrolledTeamsQuery,
  useChangeChampionshipStatus,
  useDeleteChampionship,
  useEnrollTeam,
  useUnenrollTeam,
} from './api'
import { ChampionshipForm } from './championship-form'
import { SlotsMeter } from './championships-page'

type EnrollTeamProps = { championshipId: string; enrolledIds: Set<string>; isAdmin: boolean }

function EnrollTeam({ championshipId, enrolledIds, isAdmin }: EnrollTeamProps) {
  const myTeams = useQuery({ ...myTeamsQuery, enabled: !isAdmin })
  const allTeams = useQuery({ ...teamsQuery, enabled: isAdmin })
  const enroll = useEnrollTeam(championshipId)
  const [teamId, setTeamId] = useState('')
  const candidates = isAdmin ? allTeams.data : myTeams.data
  const available = (candidates ?? []).filter((team) => !enrolledIds.has(team.id))

  if (available.length === 0) return null

  const submit = () =>
    enroll.mutate(teamId, {
      onSuccess: () => {
        toast.success('Time inscrito')
        setTeamId('')
      },
      onError: (error) => toast.error(errorMessage(error)),
    })

  return (
    <div className="mt-4 flex flex-wrap items-center gap-2 rounded-xl bg-secondary/60 p-3">
      <div className="min-w-48 flex-1">
        <NativeSelect aria-label="Time a inscrever" value={teamId} onChange={(e) => setTeamId(e.target.value)}>
          <option value="">{isAdmin ? 'Escolha um time cadastrado' : 'Escolha um dos seus times'}</option>
          {available.map((team) => (
            <option key={team.id} value={team.id}>
              {team.name}
            </option>
          ))}
        </NativeSelect>
      </div>
      <Button disabled={teamId === '' || enroll.isPending} onClick={submit}>
        Inscrever time
      </Button>
    </div>
  )
}

export function ChampionshipDetailPage({ championshipId }: { championshipId: string }) {
  const navigate = useNavigate()
  const session = useSession()
  const championship = useQuery(championshipQuery(championshipId))
  const enrolledTeams = useQuery(enrolledTeamsQuery(championshipId))
  const matches = useQuery(championshipMatchesQuery(championshipId))
  const { nameOf } = useTeamsById()
  const changeStatus = useChangeChampionshipStatus(championshipId)
  const remove = useDeleteChampionship()
  const unenroll = useUnenrollTeam(championshipId)
  const [editIsOpen, setEditIsOpen] = useState(false)
  const [matchFormIsOpen, setMatchFormIsOpen] = useState(false)
  const [deleteIsOpen, setDeleteIsOpen] = useState(false)

  const role = session?.user.role
  const isAdmin = role === 'ADMIN'

  if (championship.isPending) return <LoadingRows rows={2} />
  if (championship.isError) return <ErrorState error={championship.error} onRetry={() => championship.refetch()} />

  const data = championship.data
  const enrolledIds = new Set((enrolledTeams.data ?? []).map((team) => team.id))
  const canEnroll = (role === 'TEAM' || isAdmin) && data.status === 'OPEN'
  const teamCanBeRemoved = (ownerId: string) => data.status === 'OPEN' && (isAdmin || ownerId === session?.user.id)

  const advance = (action: 'start' | 'finish') =>
    changeStatus.mutate(action, {
      onSuccess: () => toast.success(action === 'start' ? 'Campeonato iniciado' : 'Campeonato encerrado'),
      onError: (error) => toast.error(errorMessage(error)),
    })

  const confirmDelete = () =>
    remove.mutate(championshipId, {
      onSuccess: () => {
        toast.success('Campeonato excluído')
        navigate({ to: '/championships' })
      },
      onError: (error) => {
        setDeleteIsOpen(false)
        toast.error(errorMessage(error))
      },
    })

  return (
    <>
      <Link
        to="/championships"
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeftIcon className="size-4" />
        Todos os campeonatos
      </Link>

      <PageHeader
        title={data.name}
        actions={
          isAdmin ? (
            <>
              {data.status === 'OPEN' ? (
                <Button disabled={changeStatus.isPending} onClick={() => advance('start')}>
                  <PlayIcon />
                  Iniciar campeonato
                </Button>
              ) : null}
              {data.status === 'IN_PROGRESS' ? (
                <Button disabled={changeStatus.isPending} onClick={() => advance('finish')}>
                  Encerrar campeonato
                </Button>
              ) : null}
              <Button variant="outline" size="icon" aria-label="Editar campeonato" onClick={() => setEditIsOpen(true)}>
                <PencilIcon />
              </Button>
              <Button variant="outline" size="icon" aria-label="Excluir campeonato" onClick={() => setDeleteIsOpen(true)}>
                <Trash2Icon />
              </Button>
            </>
          ) : null
        }
      />

      <div className="mb-8 max-w-sm space-y-3">
        <StatusBadge status={data.status} live={data.status === 'IN_PROGRESS'} />
        <SlotsMeter taken={data.slotsTaken} total={data.slotsTotal} />
      </div>

      <section className="mb-10">
        <h2 className="mb-3 text-xl font-semibold">Times</h2>
        {enrolledTeams.isPending ? <LoadingRows rows={2} /> : null}
        {enrolledTeams.isError ? <ErrorState error={enrolledTeams.error} /> : null}
        {enrolledTeams.isSuccess && enrolledTeams.data.length === 0 ? (
          <EmptyState
            title="Nenhum time inscrito"
            description="Os gestores de time podem se inscrever enquanto o campeonato estiver aberto."
          />
        ) : null}
        {enrolledTeams.isSuccess && enrolledTeams.data.length > 0 ? (
          <ul className="divide-y rounded-xl border bg-card">
            {enrolledTeams.data.map((team) => (
              <li key={team.id} className="flex items-center justify-between gap-3 px-4 py-3">
                <div className="min-w-0">
                  <p className="truncate font-medium">{team.name}</p>
                  <p className="truncate text-sm text-muted-foreground">{team.city}</p>
                </div>
                {teamCanBeRemoved(team.ownerId) ? (
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={unenroll.isPending}
                    onClick={() =>
                      unenroll.mutate(team.id, {
                        onSuccess: () => toast.success('Time removido do campeonato'),
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
        {canEnroll ? <EnrollTeam championshipId={championshipId} enrolledIds={enrolledIds} isAdmin={isAdmin} /> : null}
      </section>

      <section>
        <div className="mb-3 flex items-center justify-between gap-3">
          <h2 className="text-xl font-semibold">Partidas</h2>
          {isAdmin ? (
            <Button variant="outline" size="sm" onClick={() => setMatchFormIsOpen(true)}>
              <PlusIcon />
              Agendar partida
            </Button>
          ) : null}
        </div>
        {matches.isPending ? <LoadingRows rows={2} /> : null}
        {matches.isError ? <ErrorState error={matches.error} /> : null}
        {matches.isSuccess && matches.data.length === 0 ? (
          <EmptyState
            title="Nenhuma partida agendada"
            description={
              isAdmin ? 'Agende uma partida entre dois times inscritos.' : 'As partidas aparecem aqui assim que forem agendadas.'
            }
          />
        ) : null}
        {matches.isSuccess && matches.data.length > 0 ? (
          <div className="space-y-3">
            {matches.data.map((match) => (
              <MatchRow key={match.id} match={match} nameOf={nameOf} />
            ))}
          </div>
        ) : null}
      </section>

      {isAdmin ? (
        <>
          <ChampionshipForm
            key={`${data.name}-${data.slotsTotal}`}
            open={editIsOpen}
            onOpenChange={setEditIsOpen}
            championship={data}
          />
          <MatchForm
            open={matchFormIsOpen}
            onOpenChange={setMatchFormIsOpen}
            championshipId={championshipId}
            teams={enrolledTeams.data ?? []}
          />
          <ConfirmDialog
            open={deleteIsOpen}
            onOpenChange={setDeleteIsOpen}
            title="Excluir este campeonato?"
            description="Essa ação não pode ser desfeita. Um campeonato com partidas não pode ser excluído."
            confirmLabel="Excluir campeonato"
            pending={remove.isPending}
            onConfirm={confirmDelete}
          />
        </>
      ) : null}
    </>
  )
}
