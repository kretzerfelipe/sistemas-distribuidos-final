import { createFileRoute } from '@tanstack/react-router'
import { requireRole } from '@/core/auth/guards'
import { PremiumPage } from '@/modules/subscriptions/premium-page'

export const Route = createFileRoute('/_app/premium')({
  beforeLoad: () => {
    requireRole('USER', 'TEAM')
  },
  component: PremiumPage,
})
