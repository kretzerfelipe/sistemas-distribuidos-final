import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { EmptyState, ErrorState, LoadingRows } from '@/components/state'
import { PageHeader } from '@/components/page-header'
import { StatusBadge } from '@/components/status-badge'
import { errorMessage, formatDate } from '@/lib/format'
import { myInvitationsQuery, useAnswerInvitation } from './api'

export function InvitationsPage() {
  const invitations = useQuery(myInvitationsQuery)
  const answer = useAnswerInvitation()

  const respond = (id: string, choice: 'accept' | 'decline') =>
    answer.mutate(
      { id, answer: choice },
      {
        onSuccess: () => toast.success(choice === 'accept' ? 'Você entrou no time' : 'Convite recusado'),
        onError: (error) => toast.error(errorMessage(error)),
      },
    )

  return (
    <>
      <PageHeader title="Convites" description="Times que querem você no elenco. Você faz parte de um time por vez." />

      {invitations.isPending ? <LoadingRows /> : null}
      {invitations.isError ? <ErrorState error={invitations.error} onRetry={() => invitations.refetch()} /> : null}
      {invitations.isSuccess && invitations.data.length === 0 ? (
        <EmptyState
          title="Nenhum convite por enquanto"
          description="Crie seu perfil de jogador e os gestores de time poderão convidar você."
        />
      ) : null}
      {invitations.isSuccess && invitations.data.length > 0 ? (
        <ul className="divide-y rounded-xl border bg-card">
          {invitations.data.map((invitation) => (
            <li key={invitation.id} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3.5">
              <div className="min-w-0">
                <p className="truncate font-medium">{invitation.teamName}</p>
                <p className="text-sm text-muted-foreground">Recebido em {formatDate(invitation.createdAt)}</p>
              </div>
              {invitation.status === 'PENDING' ? (
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" disabled={answer.isPending} onClick={() => respond(invitation.id, 'decline')}>
                    Recusar
                  </Button>
                  <Button size="sm" disabled={answer.isPending} onClick={() => respond(invitation.id, 'accept')}>
                    Entrar no time
                  </Button>
                </div>
              ) : (
                <StatusBadge status={invitation.status} />
              )}
            </li>
          ))}
        </ul>
      ) : null}
    </>
  )
}
