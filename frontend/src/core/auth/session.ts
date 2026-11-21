import { useSyncExternalStore } from 'react'
import type { User } from '@/types'

export type Session = {
  token: string
  expiresAt: string
  user: User
}

const storageKey = 'futhub.session'
const listeners = new Set<() => void>()

function readStoredSession(): Session | null {
  try {
    const raw = localStorage.getItem(storageKey)
    if (!raw) return null
    const stored = JSON.parse(raw) as Session
    const isExpired = new Date(stored.expiresAt).getTime() <= Date.now()
    return isExpired ? null : stored
  } catch {
    return null
  }
}

let current: Session | null = readStoredSession()

function publish(next: Session | null) {
  current = next
  try {
    if (next) localStorage.setItem(storageKey, JSON.stringify(next))
    else localStorage.removeItem(storageKey)
  } catch {
    current = next
  }
  listeners.forEach((listener) => listener())
}

export function getSession(): Session | null {
  return current
}

export function getToken(): string | null {
  return current?.token ?? null
}

export function saveSession(session: Session) {
  publish(session)
}

export function clearSession() {
  publish(null)
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function useSession(): Session | null {
  return useSyncExternalStore(subscribe, getSession)
}
