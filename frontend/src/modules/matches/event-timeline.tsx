import { Trash2Icon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { eventTypeLabels } from '@/lib/format'
import type { MatchEvent } from '@/types'

type EventTimelineProps = {
  events: MatchEvent[]
  nameOf: (teamId: string) => string
  canRemove: boolean
  removing: boolean
  onRemove: (event: MatchEvent) => void
}

function eventMarker(type: MatchEvent['type']) {
  if (type === 'YELLOW_CARD') return <span className="h-4 w-3 rounded-[2px] bg-card-yellow" aria-hidden />
  if (type === 'RED_CARD') return <span className="h-4 w-3 rounded-[2px] bg-live" aria-hidden />
  const isGoal = type === 'GOAL' || type === 'OWN_GOAL'
  return <span className={cn('size-2.5 rounded-full', isGoal ? 'bg-primary' : 'bg-border')} aria-hidden />
}

function eventDetail(event: MatchEvent) {
  if (event.type === 'SUBSTITUTION') {
    const entering = event.player?.name ?? 'Desconhecido'
    const leaving = event.secondaryPlayer?.name ?? 'Desconhecido'
    return `Entra ${entering}, sai ${leaving}`
  }
  return event.player?.name ?? null
}

export function EventTimeline({ events, nameOf, canRemove, removing, onRemove }: EventTimelineProps) {
  const ordered = [...events].sort((a, b) => b.sequence - a.sequence)

  return (
    <ol className="divide-y rounded-xl border bg-card">
      {ordered.map((event) => {
        const detail = eventDetail(event)
        const isGoal = event.type === 'GOAL' || event.type === 'OWN_GOAL'
        return (
          <li key={event.id} className="grid grid-cols-[3rem_1.25rem_1fr_auto] items-center gap-3 px-4 py-3">
            <span className="font-heading text-lg font-semibold tabular-nums">{event.minute}'</span>
            <span className="flex justify-center">{eventMarker(event.type)}</span>
            <div className="min-w-0">
              <p className={cn('text-sm', isGoal ? 'font-semibold' : 'font-medium')}>
                {eventTypeLabels[event.type]}
                <span className="font-normal text-muted-foreground"> · {nameOf(event.teamId)}</span>
              </p>
              {detail ? <p className="truncate text-sm text-muted-foreground">{detail}</p> : null}
            </div>
            {canRemove ? (
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={`Remover ${eventTypeLabels[event.type]} aos ${event.minute} minutos`}
                disabled={removing}
                onClick={() => onRemove(event)}
              >
                <Trash2Icon />
              </Button>
            ) : (
              <span />
            )}
          </li>
        )
      })}
    </ol>
  )
}
