import { createFileRoute } from '@tanstack/react-router'
import { ChampionshipsPage } from '@/modules/championships/championships-page'

export const Route = createFileRoute('/_app/championships/')({
  component: ChampionshipsPage,
})
