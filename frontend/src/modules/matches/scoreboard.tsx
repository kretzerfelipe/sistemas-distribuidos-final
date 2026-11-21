import { formatDateTime, matchStatusLabels } from '@/lib/format'
import type { Match } from '@/types'

type ScoreboardProps = {
  match: Match
  homeName: string
  awayName: string
}

export function Scoreboard({ match, homeName, awayName }: ScoreboardProps) {
  const isLive = match.status === 'IN_PROGRESS'
  const hasStarted = match.status !== 'SCHEDULED'

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
        <p
          className="font-heading text-4xl font-semibold whitespace-nowrap tabular-nums sm:text-7xl"
          aria-label={hasStarted ? `Placar ${match.homeScore} a ${match.awayScore}` : 'Partida não iniciada'}
        >
          {hasStarted ? `${match.homeScore}–${match.awayScore}` : 'x'}
        </p>
        <p className="font-heading text-lg leading-tight font-semibold break-words sm:text-3xl">{awayName}</p>
      </div>
    </section>
  )
}
