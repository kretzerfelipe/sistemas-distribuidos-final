import { useSession } from '@/core/auth/session'

export function useCanWatchLive() {
  const session = useSession()
  const isAdmin = session?.user.role === 'ADMIN'
  return isAdmin
}
