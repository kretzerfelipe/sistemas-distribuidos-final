import { createFileRoute } from '@tanstack/react-router'
import { TeamDetailPage } from '@/modules/teams/team-detail-page'

export const Route = createFileRoute('/_app/teams/$teamId')({
  component: TeamRoute,
})

function TeamRoute() {
  const { teamId } = Route.useParams()
  return <TeamDetailPage teamId={teamId} />
}
