import { useQuery } from '@tanstack/react-query'
import { CheckCircle2Icon, Loader2Icon, SparklesIcon } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { ErrorState, LoadingRows } from '@/components/state'
import { PageHeader } from '@/components/page-header'
import { errorMessage, formatDateTime } from '@/lib/format'
import type { Subscription } from '@/types'
import { mySubscriptionQuery, useSubscribe } from './api'

const price = new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' })

const benefits = [
  'Acompanhe as partidas ao vivo, lance a lance, em tempo real.',
  'Receba o placar na hora em que o gol é registrado.',
  'Sem esperar o resumo que atualiza de tempos em tempos.',
]

const cancellationExplanations = {
  PAYMENT_REJECTED: 'O pagamento foi recusado, então a assinatura foi cancelada.',
  TIMEOUT: 'O pagamento demorou mais do que o permitido, então a assinatura foi cancelada.',
}

function SubscribeButton({ label }: { label: string }) {
  const subscribe = useSubscribe()
  return (
    <Button
      className="h-10"
      disabled={subscribe.isPending}
      onClick={() => subscribe.mutate(undefined, { onError: (error) => toast.error(errorMessage(error)) })}
    >
      <SparklesIcon />
      {label}
    </Button>
  )
}

function StatusCard({ subscription }: { subscription: Subscription | null }) {
  if (subscription?.status === 'ACTIVE') {
    return (
      <div className="flex items-start gap-3 rounded-xl border border-primary/30 bg-accent p-5">
        <CheckCircle2Icon className="mt-0.5 size-5 shrink-0 text-primary" />
        <div>
          <p className="font-medium">Você é premium</p>
          <p className="text-sm text-muted-foreground">
            Assinatura ativa desde {formatDateTime(subscription.createdAt)}. Abra qualquer jogo ao vivo para acompanhar em tempo real.
          </p>
        </div>
      </div>
    )
  }

  if (subscription?.status === 'PENDING') {
    return (
      <div role="status" className="flex items-start gap-3 rounded-xl border bg-card p-5">
        <Loader2Icon className="mt-0.5 size-5 shrink-0 animate-spin text-muted-foreground" />
        <div>
          <p className="font-medium">Processando o pagamento…</p>
          <p className="text-sm text-muted-foreground">Isso leva alguns segundos. Você pode ficar nesta página.</p>
        </div>
      </div>
    )
  }

  const wasClosed = subscription?.status === 'CANCELED' || subscription?.status === 'EXPIRED'
  const explanation = wasClosed && subscription.reason ? cancellationExplanations[subscription.reason] : null

  return (
    <div className="rounded-xl border bg-card p-5">
      {explanation ? <p className="mb-4 rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{explanation}</p> : null}
      <p className="font-heading text-3xl font-semibold">
        {price.format((subscription?.amountCents ?? 1990) / 100)}
        <span className="text-base font-normal text-muted-foreground"> por assinatura</span>
      </p>
      <ul className="my-4 space-y-2 text-sm">
        {benefits.map((benefit) => (
          <li key={benefit} className="flex gap-2">
            <CheckCircle2Icon className="mt-0.5 size-4 shrink-0 text-primary" />
            {benefit}
          </li>
        ))}
      </ul>
      <SubscribeButton label={wasClosed ? 'Tentar de novo' : 'Assinar o premium'} />
    </div>
  )
}

export function PremiumPage() {
  const subscription = useQuery(mySubscriptionQuery())

  return (
    <>
      <PageHeader title="Premium" description="Acompanhe os jogos ao vivo, lance a lance." />
      <div className="max-w-xl">
        {subscription.isPending ? <LoadingRows rows={1} /> : null}
        {subscription.isError ? <ErrorState error={subscription.error} onRetry={() => subscription.refetch()} /> : null}
        {subscription.isSuccess ? <StatusCard subscription={subscription.data} /> : null}
      </div>
    </>
  )
}
