import type { FieldError, Snapshot, SnapshotSummary, VerifyRequest, VerifyResult } from './types'

// ApiError 携带服务端返回的定位错误列表。
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly errors: FieldError[],
  ) {
    super(errors.map((e) => `${e.path}: ${e.message}`).join('\n') || `HTTP ${status}`)
    this.name = 'ApiError'
  }
}

async function request<T>(method: string, url: string, body?: unknown): Promise<T> {
  const res = await fetch(url, {
    method,
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    let errors: FieldError[] = [{ path: '', message: `HTTP ${res.status} ${res.statusText}` }]
    try {
      const data: unknown = await res.json()
      if (data && typeof data === 'object' && Array.isArray((data as { errors?: unknown }).errors)) {
        errors = (data as { errors: FieldError[] }).errors
      }
    } catch {
      // 保留默认错误
    }
    throw new ApiError(res.status, errors)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export const api = {
  verify: (req: VerifyRequest) => request<VerifyResult>('POST', '/api/verify', req),
  listSnapshots: () => request<SnapshotSummary[]>('GET', '/api/snapshots'),
  getSnapshot: (id: string) => request<Snapshot>('GET', `/api/snapshots/${encodeURIComponent(id)}`),
  saveSnapshot: (name: string, input: VerifyRequest, result: VerifyResult) =>
    request<Snapshot>('POST', '/api/snapshots', { name, input, result }),
  deleteSnapshot: (id: string) => request<void>('DELETE', `/api/snapshots/${encodeURIComponent(id)}`),
}
