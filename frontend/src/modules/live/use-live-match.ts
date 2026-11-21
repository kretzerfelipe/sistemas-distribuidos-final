import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { getToken } from '@/core/auth/session'
import { notificationBaseUrl } from '@/lib/api'
import type { LiveMatchState } from '@/types'
import { applyUpdate, type LiveUpdate } from './live-state'

export type LiveStatus = 'connecting' | 'live' | 'reconnecting' | 'finished' | 'unavailable'

const normalClosure = 1000
const maxFailedAttempts = 5
const firstRetryDelayMs = 1000
const maxRetryDelayMs = 10_000

function streamUrl(matchId: string, token: string) {
  const url = new URL(`${notificationBaseUrl}/matches/${matchId}/stream`, window.location.href)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  url.searchParams.set('token', token)
  return url.toString()
}

export function useLiveMatch(matchId: string, enabled: boolean) {
  const queryClient = useQueryClient()
  const [state, setState] = useState<LiveMatchState | null>(null)
  const [status, setStatus] = useState<LiveStatus>('connecting')

  useEffect(() => {
    const token = getToken()
    if (!enabled || !token) return

    let socket: WebSocket | null = null
    let retryTimer: ReturnType<typeof setTimeout> | undefined
    let failedAttempts = 0
    let isDisposed = false
    let matchFinished = false

    const finish = () => {
      matchFinished = true
      setStatus('finished')
      queryClient.invalidateQueries({ queryKey: ['matches'] })
    }

    const scheduleRetry = () => {
      failedAttempts += 1
      const tooManyFailures = failedAttempts > maxFailedAttempts
      if (tooManyFailures) {
        setStatus('unavailable')
        return
      }
      setStatus('reconnecting')
      const delay = Math.min(firstRetryDelayMs * 2 ** (failedAttempts - 1), maxRetryDelayMs)
      retryTimer = setTimeout(connect, delay)
    }

    const connect = () => {
      socket = new WebSocket(streamUrl(matchId, token))

      socket.onopen = () => {
        failedAttempts = 0
        setStatus('live')
      }

      socket.onmessage = (message) => {
        const update = JSON.parse(message.data as string) as LiveUpdate
        setState((current) => applyUpdate(current, update))
        if (update.type === 'MatchFinished') finish()
      }

      socket.onclose = (event) => {
        if (isDisposed || matchFinished) return
        if (event.code === normalClosure) {
          finish()
          return
        }
        scheduleRetry()
      }
    }

    connect()

    return () => {
      isDisposed = true
      clearTimeout(retryTimer)
      socket?.close()
    }
  }, [matchId, enabled, queryClient])

  return { state, status }
}
