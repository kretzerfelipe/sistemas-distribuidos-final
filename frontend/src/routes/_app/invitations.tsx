import { createFileRoute } from '@tanstack/react-router'
import { requireRole } from '@/core/auth/guards'
import { InvitationsPage } from '@/modules/players/invitations-page'

export const Route = createFileRoute('/_app/invitations')({
  beforeLoad: () => {
    requireRole('USER')
  },
  component: InvitationsPage,
})
