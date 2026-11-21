import { createFileRoute } from '@tanstack/react-router'
import { MatchDetailPage } from '@/modules/matches/match-detail-page'

export const Route = createFileRoute('/_app/matches/$matchId')({
  component: MatchRoute,
})

function MatchRoute() {
  const { matchId } = Route.useParams()
  return <MatchDetailPage matchId={matchId} />
}
