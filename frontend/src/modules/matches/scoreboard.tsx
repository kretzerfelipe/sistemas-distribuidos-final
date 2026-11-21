import { formatDateTime, matchStatusLabels } from '@/lib/format'
import type { Match, MatchScore } from '@/types'

type ScoreboardProps = {
  match: Match
  homeName: string
  awayName: string
  score?: MatchScore | null
}

export function Scoreboard({ match, homeName, awayName, score }: ScoreboardProps) {
  const isLive = match.status === 'IN_PROGRESS'
  const hasStarted = match.status !== 'SCHEDULED'
  const shownScore = score === undefined ? { home: match.homeScore, away: match.awayScore } : score

  let scoreText = 'x'
  let scoreLabel = 'Partida não iniciada'
  if (hasStarted && shownScore) {
    scoreText = `${shownScore.home}–${shownScore.away}`
    scoreLabel = `Placar ${shownScore.home} a ${shownScore.away}`
  } else if (hasStarted) {
    scoreText = '–'
    scoreLabel = 'Placar ainda não disponível'
  }

  return (
    <section aria-label="Placar" className="rounded-2xl bg-sidebar px-5 py-7 text-sidebar-foreground sm:px-10 sm:py-10">
      <div className="mb-6 flex items-center justify-center gap-2 text-sm text-sidebar-muted">
        {isLive ? <span className="size-2 rounded-full bg-live motion-safe:animate-pulse" /> : null}
        <span className={isLive ? 'font-medium text-sidebar-foreground' : undefined}>{matchStatusLabels[match.status]}</span>
        <span aria-hidden>·</span>
        <span>{formatDateTime(match.scheduledAt)}</span>
      </div>
      <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-3 sm:gap-8">
        <p className="text-right font-heading text-lg leading-tight font-semibold break-words sm:text-3xl">{homeName}</p>
        <p className="font-heading text-4xl font-semibold whitespace-nowrap tabular-nums sm:text-7xl" aria-label={scoreLabel}>
          {scoreText}
        </p>
        <p className="font-heading text-lg leading-tight font-semibold break-words sm:text-3xl">{awayName}</p>
      </div>
    </section>
  )
}
