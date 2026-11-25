import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { EmptyState, ErrorState, LoadingRows } from '@/components/state'
import { PageHeader } from '@/components/page-header'
import { cn } from '@/lib/utils'
import { errorMessage, formatDateTime } from '@/lib/format'
import { notificationsQuery, useMarkAsRead } from './api'

export function NotificationsPage() {
  const notifications = useQuery(notificationsQuery())
  const markAsRead = useMarkAsRead()

  return (
    <>
      <PageHeader title="Notificações" description="Avisos sobre a sua assinatura." />

      {notifications.isPending ? <LoadingRows /> : null}
      {notifications.isError ? <ErrorState error={notifications.error} onRetry={() => notifications.refetch()} /> : null}
      {notifications.isSuccess && notifications.data.length === 0 ? (
        <EmptyState title="Nenhuma notificação" description="Quando a sua assinatura mudar, o aviso aparece aqui." />
      ) : null}
      {notifications.isSuccess && notifications.data.length > 0 ? (
        <ul className="divide-y overflow-hidden rounded-xl border bg-card">
          {notifications.data.map((notification) => (
            <li key={notification.id} className="flex items-start justify-between gap-3 px-4 py-3.5">
              <div className="min-w-0">
                <p className={cn('text-sm', notification.read ? 'text-muted-foreground' : 'font-medium')}>{notification.message}</p>
                <p className="mt-0.5 text-xs text-muted-foreground">{formatDateTime(notification.createdAt)}</p>
              </div>
              {notification.read ? null : (
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={markAsRead.isPending}
                  onClick={() => markAsRead.mutate(notification.id, { onError: (error) => toast.error(errorMessage(error)) })}
                >
                  Marcar como lida
                </Button>
              )}
            </li>
          ))}
        </ul>
      ) : null}
    </>
  )
}
