import type { LiveMatchState, MatchEvent, MatchScore } from '@/types'

export type LiveUpdate = {
  type: string
  matchId: string
  sequence: number
  payload: unknown
}

type RegisteredPayload = { event: MatchEvent; score: MatchScore }
type RemovedPayload = { eventId: string; score: MatchScore }

export function applyUpdate(current: LiveMatchState | null, update: LiveUpdate): LiveMatchState | null {
  if (update.type === 'MatchState') return update.payload as LiveMatchState

  const hasNoBaseState = current === null
  if (hasNoBaseState) return current

  if (update.type === 'MatchEventRegistered') {
    const { event, score } = update.payload as RegisteredPayload
    const alreadyKnown = current.events.some((known) => known.id === event.id)
    const events = alreadyKnown ? current.events : [...current.events, event]
    return { ...current, events, score, sequence: update.sequence }
  }

  if (update.type === 'MatchEventRemoved') {
    const { eventId, score } = update.payload as RemovedPayload
    const events = current.events.filter((known) => known.id !== eventId)
    return { ...current, events, score, sequence: update.sequence }
  }

  if (update.type === 'MatchFinished') {
    const { score } = update.payload as { score: MatchScore }
    return { ...current, score, sequence: update.sequence }
  }

  return current
}
