import type { ExecResponse, TableInfo } from './types'

/** バックエンドAPIのベースURL。ローカル開発ではViteのプロキシ経由で同一
 * オリジン扱いになるため空文字(相対パス)のままでよいが、本番/ステージング
 * ではフロントエンド(CloudFront)とバックエンド(EC2)が別オリジンになるため、
 * ビルド時に環境変数VITE_API_BASE_URLで絶対URLを指定する。 */
const API_BASE = import.meta.env.VITE_API_BASE_URL ?? ''

/** SQLを1文実行する。Cookieセッションで状態が引き継がれる(credentials: 'include')。
 * tableを渡すと、レスポンスにそのテーブルのB+Treeスナップショットが含まれる。
 * wal=trueを渡すと、レスポンスにこのセッションのWALレコード一覧が含まれる。 */
export async function execSql(sql: string, table?: string, wal?: boolean): Promise<ExecResponse> {
  const res = await fetch(`${API_BASE}/api/exec`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ sql, table, wal }),
  })
  if (!res.ok) {
    throw new Error(`/api/exec failed: ${res.status}`)
  }
  return (await res.json()) as ExecResponse
}

/** セッション内に存在する全テーブルの名前・カラム一覧を取得する(テーブル切り替えタブ用)。 */
export async function listTables(): Promise<TableInfo[]> {
  const res = await fetch(`${API_BASE}/api/tables`, { credentials: 'include' })
  if (!res.ok) {
    throw new Error(`/api/tables failed: ${res.status}`)
  }
  return (await res.json()) as TableInfo[]
}

/** 今のセッションのDBを閉じて空の状態から作り直す。 */
export async function resetSession(): Promise<void> {
  const res = await fetch(`${API_BASE}/api/reset`, { method: 'POST', credentials: 'include' })
  if (!res.ok) {
    throw new Error(`/api/reset failed: ${res.status}`)
  }
}
