import { redirect } from '@tanstack/react-router'
import { getSession } from '@/core/auth/session'
import type { Role } from '@/types'

export function requireSession() {
  const session = getSession()
  if (!session) throw redirect({ to: '/login' })
  return session
}

export function requireRole(...roles: Role[]) {
  const session = requireSession()
  const isAllowed = roles.includes(session.user.role)
  if (!isAllowed) throw redirect({ to: '/' })
  return session
}

export function redirectIfSignedIn() {
  if (getSession()) throw redirect({ to: '/' })
}
