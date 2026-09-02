import type { Battle, Monster, QueryResult } from './types'

/** APIエラー。サーバーが返したメッセージをそのまま持つ。 */
export class ApiError extends Error {}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
  })
  const text = await res.text()
  const body = text ? JSON.parse(text) : null

  if (!res.ok) {
    throw new ApiError(body?.error ?? `リクエストに失敗しました (${res.status})`)
  }
  return body as T
}

export const api = {
  monsters: () => request<{ monsters: Monster[] }>('/api/monsters').then((r) => r.monsters),

  startBattle: (monsterId: number) =>
    request<Battle>('/api/battles', {
      method: 'POST',
      body: JSON.stringify({ monster_id: monsterId }),
    }),

  battle: (id: string) => request<Battle>(`/api/battles/${id}`),

  /** 現在のフェーズに応じてSQLを実行する。SQL自体のエラーは例外ではなくresult.errorで返る。 */
  runQuery: (id: string, sql: string) =>
    request<QueryResult>(`/api/battles/${id}/query`, {
      method: 'POST',
      body: JSON.stringify({ sql }),
    }),

  /** 分析フェーズから次のフェーズへ進む。 */
  advance: (id: string) => request<Battle>(`/api/battles/${id}/advance`, { method: 'POST' }),

  quit: (id: string) => request<Battle>(`/api/battles/${id}/quit`, { method: 'POST' }),

  restart: (id: string) => request<Battle>(`/api/battles/${id}/restart`, { method: 'POST' }),
}
