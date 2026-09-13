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
  verify: async (req: VerifyRequest) => normalizeResult(await request<VerifyResult>('POST', '/api/verify', req)),
  listSnapshots: () => request<SnapshotSummary[]>('GET', '/api/snapshots'),
  getSnapshot: async (id: string) => {
    const snap = await request<Snapshot>('GET', `/api/snapshots/${encodeURIComponent(id)}`)
    snap.result = normalizeResult(snap.result)
    return snap
  },
  saveSnapshot: (name: string, input: VerifyRequest, result: VerifyResult) =>
    request<Snapshot>('POST', '/api/snapshots', { name, input, result }),
  deleteSnapshot: (id: string) => request<void>('DELETE', `/api/snapshots/${encodeURIComponent(id)}`),
}

// normalizeResult 把结果中可能为 null 的数组字段归一化为 []，
// 避免历史快照或异常响应让渲染层按数组访问时崩溃。
function normalizeResult(r: VerifyResult): VerifyResult {
  r.pipes ??= []
  r.states ??= []
  r.traversed ??= []
  r.pruned ??= []
  r.unmapped ??= []
  for (const s of r.states) s.couplers ??= []
  for (const p of r.pipes) p.source.couplers ??= []
  for (const p of r.pruned) p.couplers ??= []
  for (const u of r.unmapped) u.couplers ??= []
  return r
}
