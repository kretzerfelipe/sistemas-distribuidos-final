import { createFileRoute } from '@tanstack/react-router'
import { ChampionshipDetailPage } from '@/modules/championships/championship-detail-page'

export const Route = createFileRoute('/_app/championships/$championshipId')({
  component: ChampionshipRoute,
})

function ChampionshipRoute() {
  const { championshipId } = Route.useParams()
  return <ChampionshipDetailPage championshipId={championshipId} />
}
