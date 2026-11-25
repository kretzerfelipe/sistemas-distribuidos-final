import { Link } from '@tanstack/react-router'
import { LockIcon } from 'lucide-react'
import { EmptyState, ErrorState, LoadingRows } from '@/components/state'
import { eventTypeLabels } from '@/lib/format'
import type { LiveSummary } from '@/types'

type SummaryPanelProps = {
  summary: LiveSummary | undefined
  isPending: boolean
  error: unknown
  nameOf: (teamId: string) => string
}

const timeFormat = new Intl.DateTimeFormat('pt-BR', { timeStyle: 'short' })

export function SummaryPanel({ summary, isPending, error, nameOf }: SummaryPanelProps) {
  return (
    <section>
      <div className="mb-4 flex items-start gap-3 rounded-xl border bg-card p-4">
        <LockIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
        <div className="text-sm">
          <p className="font-medium">Acompanhamento em tempo real é exclusivo para assinantes</p>
          <p className="text-muted-foreground">
            Você vê um resumo da partida com os lances principais, atualizado de tempos em tempos.
          </p>
          <Link to="/premium" className="mt-1 inline-block font-medium text-primary underline underline-offset-4">
            Conhecer o premium
          </Link>
        </div>
      </div>

      <h2 className="mb-3 text-xl font-semibold">Resumo da partida</h2>
      {isPending ? <LoadingRows rows={2} /> : null}
      {error ? <ErrorState error={error} /> : null}
      {summary && summary.highlights.length === 0 ? (
        <EmptyState title="Nenhum lance de destaque ainda" description="Gols e cartões aparecem aqui." />
      ) : null}
      {summary && summary.highlights.length > 0 ? (
        <ol className="divide-y rounded-xl border bg-card">
          {[...summary.highlights]
            .sort((a, b) => b.sequence - a.sequence)
            .map((event) => (
              <li key={event.id} className="flex items-center gap-3 px-4 py-3">
                <span className="w-10 font-heading text-lg font-semibold tabular-nums">{event.minute}'</span>
                <div className="min-w-0">
                  <p className="text-sm font-medium">
                    {eventTypeLabels[event.type]}
                    <span className="font-normal text-muted-foreground"> · {nameOf(event.teamId)}</span>
                  </p>
                  {event.player ? <p className="truncate text-sm text-muted-foreground">{event.player.name}</p> : null}
                </div>
              </li>
            ))}
        </ol>
      ) : null}
      {summary ? (
        <p className="mt-3 text-xs text-muted-foreground">
          Resumo gerado às {timeFormat.format(new Date(summary.generatedAt))}.
        </p>
      ) : null}
    </section>
  )
}
