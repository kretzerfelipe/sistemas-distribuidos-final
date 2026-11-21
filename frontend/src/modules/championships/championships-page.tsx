import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { PlusIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { EmptyState, ErrorState, LoadingRows } from '@/components/state'
import { PageHeader } from '@/components/page-header'
import { StatusBadge } from '@/components/status-badge'
import { useSession } from '@/core/auth/session'
import { formatDate } from '@/lib/format'
import type { Championship } from '@/types'
import { championshipsQuery } from './api'
import { ChampionshipForm } from './championship-form'

export function SlotsMeter({ taken, total }: { taken: number; total: number }) {
  const percentage = total === 0 ? 0 : Math.min(100, (taken / total) * 100)
  return (
    <div>
      <div className="h-1.5 overflow-hidden rounded-full bg-secondary">
        <div className="h-full rounded-full bg-primary" style={{ width: `${percentage}%` }} />
      </div>
      <p className="mt-1.5 text-xs text-muted-foreground tabular-nums">
        {taken} de {total} times inscritos
      </p>
    </div>
  )
}

function ChampionshipCard({ championship }: { championship: Championship }) {
  return (
    <Link
      to="/championships/$championshipId"
      params={{ championshipId: championship.id }}
      className="group block rounded-xl border bg-card p-5 transition-colors hover:border-primary/40 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
    >
      <div className="flex items-start justify-between gap-3">
        <h2 className="font-heading text-lg leading-snug font-semibold">{championship.name}</h2>
        <StatusBadge status={championship.status} live={championship.status === 'IN_PROGRESS'} />
      </div>
      <div className="mt-5">
        <SlotsMeter taken={championship.slotsTaken} total={championship.slotsTotal} />
      </div>
      <p className="mt-4 text-xs text-muted-foreground">Criado em {formatDate(championship.createdAt)}</p>
    </Link>
  )
}

export function ChampionshipsPage() {
  const session = useSession()
  const championships = useQuery(championshipsQuery)
  const [formIsOpen, setFormIsOpen] = useState(false)
  const isAdmin = session?.user.role === 'ADMIN'

  return (
    <>
      <PageHeader
        title="Campeonatos"
        description="Acompanhe os torneios, veja quem está inscrito e abra qualquer partida."
        actions={
          isAdmin ? (
            <Button onClick={() => setFormIsOpen(true)}>
              <PlusIcon />
              Novo campeonato
            </Button>
          ) : null
        }
      />

      {championships.isPending ? <LoadingRows /> : null}
      {championships.isError ? <ErrorState error={championships.error} onRetry={() => championships.refetch()} /> : null}
      {championships.isSuccess && championships.data.length === 0 ? (
        <EmptyState
          title="Nenhum campeonato ainda"
          description={isAdmin ? 'Crie o primeiro para os times se inscreverem.' : 'Um administrador vai criar o primeiro em breve.'}
          action={isAdmin ? <Button onClick={() => setFormIsOpen(true)}>Novo campeonato</Button> : undefined}
        />
      ) : null}
      {championships.isSuccess && championships.data.length > 0 ? (
        <div className="grid gap-4 sm:grid-cols-2">
          {championships.data.map((championship) => (
            <ChampionshipCard key={championship.id} championship={championship} />
          ))}
        </div>
      ) : null}

      {isAdmin ? <ChampionshipForm open={formIsOpen} onOpenChange={setFormIsOpen} /> : null}
    </>
  )
}
