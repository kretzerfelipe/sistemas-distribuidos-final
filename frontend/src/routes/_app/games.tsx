import { createFileRoute } from '@tanstack/react-router'
import { GamesPage } from '@/modules/matches/games-page'

export const Route = createFileRoute('/_app/games')({
  component: GamesPage,
})
