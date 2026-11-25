import { clearSession, getToken } from '@/core/auth/session'

export const mainBaseUrl = import.meta.env.VITE_API_URL ?? '/api'
export const notificationBaseUrl = import.meta.env.VITE_NOTIFICATION_URL ?? '/notification'

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
  file?: File
  signal?: AbortSignal
}

type ErrorBody = { error?: { code?: string; message?: string } }

function createApi(baseUrl: string) {
  return async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const { method = 'GET', body, file, signal } = options
    const token = getToken()

    const headers: Record<string, string> = {}
    if (body !== undefined) headers['Content-Type'] = 'application/json'
    if (file) headers['Content-Type'] = file.type
    if (token) headers.Authorization = `Bearer ${token}`

    let response: Response
    try {
      response = await fetch(`${baseUrl}${path}`, {
        method,
        headers,
        body: file ?? (body === undefined ? undefined : JSON.stringify(body)),
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
      throw new ApiError(response.status, detail?.code ?? 'unknown_error', detail?.message ?? '')
    }

    return payload as T
  }
}

export const api = createApi(mainBaseUrl)
export const notificationApi = createApi(notificationBaseUrl)
