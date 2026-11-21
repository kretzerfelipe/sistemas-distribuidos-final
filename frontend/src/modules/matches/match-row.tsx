import { Link } from '@tanstack/react-router'
import { StatusBadge } from '@/components/status-badge'
import { formatDateTime } from '@/lib/format'
import type { Match } from '@/types'

export function MatchRow({ match, nameOf }: { match: Match; nameOf: (teamId: string) => string }) {
  const hasStarted = match.status !== 'SCHEDULED'
  return (
    <Link
      to="/matches/$matchId"
      params={{ matchId: match.id }}
      className="grid grid-cols-[1fr_auto_1fr] items-center gap-3 rounded-xl border bg-card px-4 py-3.5 transition-colors hover:border-primary/40 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
    >
      <span className="truncate text-right font-medium">{nameOf(match.homeTeamId)}</span>
      <span className="min-w-20 text-center">
        {hasStarted ? (
          <span className="font-heading text-xl font-semibold tabular-nums">
            {match.homeScore} – {match.awayScore}
          </span>
        ) : (
          <span className="text-sm text-muted-foreground">x</span>
        )}
      </span>
      <span className="truncate font-medium">{nameOf(match.awayTeamId)}</span>
      <span className="col-span-3 flex items-center justify-between gap-2 border-t pt-2.5 text-xs text-muted-foreground">
        <span>{formatDateTime(match.scheduledAt)}</span>
        <StatusBadge status={match.status} live={match.status === 'IN_PROGRESS'} />
      </span>
    </Link>
  )
}
