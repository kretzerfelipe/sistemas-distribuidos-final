import { createFileRoute } from '@tanstack/react-router'
import { requireSession } from '@/core/auth/guards'
import { AppShell } from '@/core/layout/app-shell'

export const Route = createFileRoute('/_app')({
  beforeLoad: () => {
    requireSession()
  },
  component: AppShell,
})
