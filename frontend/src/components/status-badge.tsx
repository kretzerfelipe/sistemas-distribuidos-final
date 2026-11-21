import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { championshipStatusLabels, invitationStatusLabels, matchStatusLabels } from '@/lib/format'
import type { ChampionshipStatus, InvitationStatus, MatchStatus } from '@/types'

type Status = ChampionshipStatus | MatchStatus | InvitationStatus

const labels: Record<string, string> = {
  ...championshipStatusLabels,
  ...matchStatusLabels,
  ...invitationStatusLabels,
}

const tones: Record<string, string> = {
  OPEN: 'bg-accent text-accent-foreground',
  SCHEDULED: 'bg-secondary text-secondary-foreground',
  PENDING: 'bg-card-yellow/40 text-foreground',
  ACCEPTED: 'bg-accent text-accent-foreground',
  FINISHED: 'bg-muted text-muted-foreground',
  DECLINED: 'bg-muted text-muted-foreground',
  CANCELED: 'bg-muted text-muted-foreground',
  IN_PROGRESS: 'bg-primary text-primary-foreground',
}

export function StatusBadge({ status, live = false }: { status: Status; live?: boolean }) {
  return (
    <Badge variant="secondary" className={cn('gap-1.5 font-medium', tones[status])}>
      {live ? <span className="size-1.5 rounded-full bg-live motion-safe:animate-pulse" /> : null}
      {labels[status]}
    </Badge>
  )
}
