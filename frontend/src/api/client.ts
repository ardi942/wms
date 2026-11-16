const BASE_URL = import.meta.env.VITE_API_URL ?? '/api/v1'
const TOKEN_KEY = 'wms.token'

function createIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  const bytes = crypto.getRandomValues(new Uint8Array(16))
  bytes[6] = (bytes[6] & 0x0f) | 0x40
  bytes[8] = (bytes[8] & 0x3f) | 0x80
  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}

export class ApiError extends Error {
  status: number
  code: string
  details: Record<string, unknown> | null

  constructor(status: number, code: string, message: string, details: Record<string, unknown> | null = null) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
  }
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

export function toQuery(params?: Record<string, unknown>): string {
  if (!params) return ''
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    search.set(key, String(value))
  }
  const raw = search.toString()
  return raw ? `?${raw}` : ''
}

interface RequestOptions {
  method?: string
  body?: unknown
  formData?: FormData
  idempotencyKey?: string
  signal?: AbortSignal
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const method = options.method ?? 'GET'
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (options.body !== undefined && options.formData === undefined) headers['Content-Type'] = 'application/json'
  const token = getToken()
  if (token) headers.Authorization = `Bearer ${token}`
  if ((method === 'POST' || method === 'PATCH') && options.formData === undefined) {
    headers['Idempotency-Key'] = options.idempotencyKey ?? createIdempotencyKey()
  }
  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers,
    body: options.formData ?? (options.body === undefined ? undefined : JSON.stringify(options.body)),
    signal: options.signal,
  })
  if (res.status === 204) return undefined as T
  const text = await res.text()
  let payload: unknown = null
  if (text) {
    try {
      payload = JSON.parse(text)
    } catch {
      payload = null
    }
  }
  if (!res.ok) {
    const error = (payload as { error?: { code?: string; message?: string; details?: Record<string, unknown> } } | null)?.error
    if (res.status === 401) {
      setToken(null)
      window.dispatchEvent(new CustomEvent('wms:unauthorized'))
    }
    throw new ApiError(
      res.status,
      error?.code ?? 'REQUEST_FAILED',
      error?.message ?? `Request failed with status ${res.status}`,
      error?.details ?? null,
    )
  }
  return payload as T
}

export const api = {
  get: <T>(path: string, signal?: AbortSignal) => request<T>(path, { signal }),
  post: <T>(path: string, body?: unknown, idempotencyKey?: string) =>
    request<T>(path, { method: 'POST', body, idempotencyKey }),
  patch: <T>(path: string, body?: unknown, idempotencyKey?: string) =>
    request<T>(path, { method: 'PATCH', body, idempotencyKey }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
  upload: <T>(path: string, formData: FormData) => request<T>(path, { method: 'POST', formData }),
}

function handleUnauthorized(res: Response) {
  if (res.status === 401) {
    setToken(null)
    window.dispatchEvent(new CustomEvent('wms:unauthorized'))
  }
}

export async function fetchBlob(path: string, signal?: AbortSignal): Promise<Blob> {
  const token = getToken()
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    signal,
  })
  if (!res.ok) {
    handleUnauthorized(res)
    let message = `Request failed with status ${res.status}`
    let code = 'REQUEST_FAILED'
    let details: Record<string, unknown> | null = null
    try {
      const payload = await res.json()
      message = payload?.error?.message ?? message
      code = payload?.error?.code ?? code
      details = payload?.error?.details ?? null
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, code, message, details)
  }
  return res.blob()
}

export async function downloadCsv(path: string, params?: Record<string, unknown>): Promise<Blob> {
  const token = getToken()
  const res = await fetch(`${BASE_URL}${path}${toQuery(params)}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) {
    let message = `Export failed with status ${res.status}`
    try {
      const payload = await res.json()
      message = payload?.error?.message ?? message
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, 'EXPORT_FAILED', message)
  }
  return res.blob()
}

export function saveBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
