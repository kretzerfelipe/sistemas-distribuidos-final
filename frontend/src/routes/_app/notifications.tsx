import { createFileRoute } from '@tanstack/react-router'
import { requireRole } from '@/core/auth/guards'
import { NotificationsPage } from '@/modules/notifications/notifications-page'

export const Route = createFileRoute('/_app/notifications')({
  beforeLoad: () => {
    requireRole('USER', 'TEAM')
  },
  component: NotificationsPage,
})
