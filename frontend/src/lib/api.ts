import { clearSession, getToken } from '@/core/auth/session'

const baseUrl = import.meta.env.VITE_API_URL ?? '/api'

export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

type RequestOptions = {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  signal?: AbortSignal
}

type ErrorBody = { error?: { code?: string; message?: string } }

export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, signal } = options
  const token = getToken()

  const headers: Record<string, string> = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (token) headers.Authorization = `Bearer ${token}`

  let response: Response
  try {
    response = await fetch(`${baseUrl}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
    })
  } catch (cause) {
    if (cause instanceof DOMException && cause.name === 'AbortError') throw cause
    throw new ApiError(0, 'network_error', '')
  }

  const hasBody = response.status !== 204 && response.headers.get('content-length') !== '0'
  const payload = hasBody ? await response.json().catch(() => null) : null

  if (!response.ok) {
    const detail = (payload as ErrorBody | null)?.error
    const tokenWasRejected = response.status === 401 && token !== null
    if (tokenWasRejected) clearSession()
    throw new ApiError(
      response.status,
      detail?.code ?? 'unknown_error',
      detail?.message ?? '',
    )
  }

  return payload as T
}
