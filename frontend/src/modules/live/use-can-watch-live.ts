import { useQuery } from '@tanstack/react-query'
import { useSession } from '@/core/auth/session'
import { mySubscriptionQuery } from '@/modules/subscriptions/api'

export function useCanWatchLive() {
  const session = useSession()
  const isAdmin = session?.user.role === 'ADMIN'
  const subscription = useQuery(mySubscriptionQuery(!isAdmin && session !== null))
  const hasActiveSubscription = subscription.data?.status === 'ACTIVE'
  return isAdmin || hasActiveSubscription
}
