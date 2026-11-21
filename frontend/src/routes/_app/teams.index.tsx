import { createFileRoute } from '@tanstack/react-router'
import { TeamsPage } from '@/modules/teams/teams-page'

export const Route = createFileRoute('/_app/teams/')({
  component: TeamsPage,
})
