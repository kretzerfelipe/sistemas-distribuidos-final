import { createFileRoute } from '@tanstack/react-router'
import { requireRole } from '@/core/auth/guards'
import { UsersPage } from '@/modules/users/users-page'

export const Route = createFileRoute('/_app/users')({
  beforeLoad: () => {
    requireRole('ADMIN')
  },
  component: UsersPage,
})
