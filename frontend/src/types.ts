export type Role = 'ADMIN' | 'TEAM' | 'USER'

export type User = {
  id: string
  name: string
  email: string
  role: Role
  createdAt: string
}

export type LoginResult = {
  accessToken: string
  tokenType: string
  expiresAt: string
  user: User
}

export type Position = 'GOALKEEPER' | 'DEFENDER' | 'MIDFIELDER' | 'FORWARD'

export type Player = {
  id: string
  name: string
  position: Position
  shirtNumber: number
  createdAt: string
}

export type Team = {
  id: string
  ownerId: string
  name: string
  city: string
  createdAt: string
}

export type InvitationStatus = 'PENDING' | 'ACCEPTED' | 'DECLINED' | 'CANCELED'

export type Invitation = {
  id: string
  teamId: string
  teamName: string
  playerId: string
  status: InvitationStatus
  createdAt: string
}

export type ChampionshipStatus = 'OPEN' | 'IN_PROGRESS' | 'FINISHED'

export type Championship = {
  id: string
  name: string
  slotsTotal: number
  slotsTaken: number
  status: ChampionshipStatus
  createdAt: string
}

export type MatchStatus = 'SCHEDULED' | 'IN_PROGRESS' | 'FINISHED'

export type Match = {
  id: string
  championshipId: string
  homeTeamId: string
  awayTeamId: string
  scheduledAt: string
  status: MatchStatus
  homeScore: number
  awayScore: number
  createdAt: string
}

export type MatchWindow = {
  from: string
  to: string
  hasMore: boolean
  matches: Match[]
}

export type MatchEventType =
  | 'GOAL'
  | 'OWN_GOAL'
  | 'SHOT'
  | 'SAVE'
  | 'FOUL'
  | 'YELLOW_CARD'
  | 'RED_CARD'
  | 'SUBSTITUTION'
  | 'CORNER'
  | 'OFFSIDE'

export type NamedReference = { id: string; name: string }

export type MatchScore = { home: number; away: number }

export type LiveMatchState = {
  matchId: string
  sequence: number
  championship: NamedReference
  homeTeam: NamedReference
  awayTeam: NamedReference
  scheduledAt: string
  startedAt: string
  score: MatchScore
  events: MatchEvent[]
}

export type LiveSummary = {
  matchId: string
  sequence: number
  championship: NamedReference
  homeTeam: NamedReference
  awayTeam: NamedReference
  startedAt: string
  score: MatchScore
  highlights: MatchEvent[]
  generatedAt: string
}

export type MatchEvent = {
  id: string
  sequence: number
  type: MatchEventType
  minute: number
  teamId: string
  player: NamedReference | null
  secondaryPlayer: NamedReference | null
  registeredAt: string
}
