import { createFileRoute } from '@tanstack/react-router'
import { requireRole } from '@/core/auth/guards'
import { PlayerPage } from '@/modules/players/player-page'

export const Route = createFileRoute('/_app/player')({
  beforeLoad: () => {
    requireRole('USER')
  },
  component: PlayerPage,
})
