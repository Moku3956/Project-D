import type { ExecResponse } from './types'

/** SQLを1文実行する。Cookieセッションで状態が引き継がれる(credentials: 'include')。
 * tableを渡すと、レスポンスにそのテーブルのB+Treeスナップショットが含まれる。 */
export async function execSql(sql: string, table?: string): Promise<ExecResponse> {
  const res = await fetch('/api/exec', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ sql, table }),
  })
  if (!res.ok) {
    throw new Error(`/api/exec failed: ${res.status}`)
  }
  return (await res.json()) as ExecResponse
}

/** 今のセッションのDBを閉じて空の状態から作り直す。 */
export async function resetSession(): Promise<void> {
  const res = await fetch('/api/reset', { method: 'POST', credentials: 'include' })
  if (!res.ok) {
    throw new Error(`/api/reset failed: ${res.status}`)
  }
}
