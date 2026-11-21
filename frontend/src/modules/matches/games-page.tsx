import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { StatusBadge } from '@/components/status-badge'
import { EmptyState, ErrorState, LoadingRows } from '@/components/state'
import { PageHeader } from '@/components/page-header'
import { championshipsQuery } from '@/modules/championships/api'
import { useTeamsById } from '@/modules/teams/use-teams-by-id'
import type { Match } from '@/types'
import { liveMatchesQuery, matchWindowsQuery, startOfToday } from './api'

type DayGroup = { key: string; date: Date; matches: Match[] }

const timeFormat = new Intl.DateTimeFormat('pt-BR', { timeStyle: 'short' })
const dayFormat = new Intl.DateTimeFormat('pt-BR', { weekday: 'long', day: 'numeric', month: 'long' })

function dayKey(date: Date) {
  return `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`
}

function dayTitle(date: Date) {
  const today = new Date()
  const tomorrow = new Date()
  tomorrow.setDate(today.getDate() + 1)
  if (dayKey(date) === dayKey(today)) return 'Hoje'
  if (dayKey(date) === dayKey(tomorrow)) return 'Amanhã'
  return dayFormat.format(date)
}

function groupByDay(matches: Match[]): DayGroup[] {
  const groups = new Map<string, DayGroup>()
  for (const match of matches) {
    const date = new Date(match.scheduledAt)
    const key = dayKey(date)
    const group = groups.get(key) ?? { key, date, matches: [] }
    group.matches.push(match)
    groups.set(key, group)
  }
  return [...groups.values()]
    .map((group) => ({
      ...group,
      matches: group.matches.sort((a, b) => new Date(a.scheduledAt).getTime() - new Date(b.scheduledAt).getTime()),
    }))
    .sort((a, b) => a.date.getTime() - b.date.getTime())
}

type GameRowProps = {
  match: Match
  nameOf: (teamId: string) => string
  championshipName: string
}

function GameRow({ match, nameOf, championshipName }: GameRowProps) {
  const isLive = match.status === 'IN_PROGRESS'
  const hasStarted = match.status !== 'SCHEDULED'

  return (
    <Link
      to="/matches/$matchId"
      params={{ matchId: match.id }}
      className="grid grid-cols-[3.5rem_1fr_auto] items-center gap-3 px-4 py-3.5 transition-colors hover:bg-secondary/60 focus-visible:bg-secondary/60 focus-visible:outline-none"
    >
      <span className="font-heading text-lg font-semibold tabular-nums">{timeFormat.format(new Date(match.scheduledAt))}</span>
      <div className="min-w-0">
        <p className="flex items-center gap-2 font-medium">
          <span className="truncate">{nameOf(match.homeTeamId)}</span>
          <span className="shrink-0 font-heading tabular-nums">
            {hasStarted ? `${match.homeScore} – ${match.awayScore}` : 'x'}
          </span>
          <span className="truncate">{nameOf(match.awayTeamId)}</span>
        </p>
        <p className="truncate text-sm text-muted-foreground">{championshipName}</p>
      </div>
      <StatusBadge status={match.status} live={isLive} />
    </Link>
  )
}

function useLoadMoreOnScroll(onReachEnd: () => void, canLoadMore: boolean, pagesLoaded: number) {
  const sentinel = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const element = sentinel.current
    if (!element || !canLoadMore) return

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) onReachEnd()
      },
      { rootMargin: '200px' },
    )
    observer.observe(element)
    return () => observer.disconnect()
  }, [onReachEnd, canLoadMore, pagesLoaded])

  return sentinel
}

export function GamesPage() {
  const [firstDay] = useState(startOfToday)
  const windows = useInfiniteQuery(matchWindowsQuery(firstDay))
  const live = useQuery(liveMatchesQuery)
  const championships = useQuery(championshipsQuery)
  const { nameOf } = useTeamsById()

  const championshipName = (id: string) => championships.data?.find((item) => item.id === id)?.name ?? 'Campeonato'

  const liveMatches = useMemo(() => live.data ?? [], [live.data])
  const liveIds = useMemo(() => new Set(liveMatches.map((match) => match.id)), [liveMatches])
  const days = useMemo(() => {
    const scheduled = (windows.data?.pages ?? []).flatMap((window) => window.matches)
    return groupByDay(scheduled.filter((match) => !liveIds.has(match.id)))
  }, [windows.data, liveIds])

  const canLoadMore = windows.hasNextPage && !windows.isFetchingNextPage && !windows.isError
  const sentinel = useLoadMoreOnScroll(() => windows.fetchNextPage(), canLoadMore, windows.data?.pages.length ?? 0)

  const hasAnyGame = liveMatches.length > 0 || days.length > 0
  const reachedTheEnd = windows.isSuccess && !windows.hasNextPage

  return (
    <>
      <PageHeader title="Jogos" description="As partidas de hoje em diante, dia a dia. Role para ver os próximos dias." />

      {liveMatches.length > 0 ? (
        <section className="mb-8">
          <h2 className="mb-3 flex items-center gap-2 text-xl font-semibold">
            <span className="size-2 rounded-full bg-live motion-safe:animate-pulse" />
            Ao vivo agora
          </h2>
          <div className="divide-y overflow-hidden rounded-xl border border-live/40 bg-card">
            {liveMatches.map((match) => (
              <GameRow key={match.id} match={match} nameOf={nameOf} championshipName={championshipName(match.championshipId)} />
            ))}
          </div>
        </section>
      ) : null}

      {windows.isPending ? <LoadingRows /> : null}
      {windows.isError ? <ErrorState error={windows.error} onRetry={() => windows.refetch()} /> : null}

      {days.map((day) => (
        <section key={day.key} className="mb-8">
          <h2 className="mb-3 text-xl font-semibold capitalize first-letter:uppercase">{dayTitle(day.date)}</h2>
          <div className="divide-y overflow-hidden rounded-xl border bg-card">
            {day.matches.map((match) => (
              <GameRow key={match.id} match={match} nameOf={nameOf} championshipName={championshipName(match.championshipId)} />
            ))}
          </div>
        </section>
      ))}

      {windows.isSuccess && !hasAnyGame && reachedTheEnd ? (
        <EmptyState title="Nenhum jogo agendado" description="Quando um administrador agendar partidas, elas aparecem aqui." />
      ) : null}

      <div ref={sentinel} aria-hidden className="h-px" />
      {windows.isFetchingNextPage ? <p className="py-4 text-center text-sm text-muted-foreground">Carregando mais jogos…</p> : null}
      {reachedTheEnd && hasAnyGame ? (
        <p className="py-4 text-center text-sm text-muted-foreground">Não há mais jogos agendados.</p>
      ) : null}
    </>
  )
}
