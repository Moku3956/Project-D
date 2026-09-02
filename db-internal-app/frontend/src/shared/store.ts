import { create } from 'zustand'
import { execSql, resetSession } from './api'
import type { ExecResponse, TreeSnapshot } from './types'

const DEFAULT_TABLE = 'users'
const DEFAULT_SQL = `CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))`

type State = {
  sql: string
  table: string
  busy: boolean
  error: string | null
  lastResult: ExecResponse | null
  tree: TreeSnapshot | null
  /** 直前のスナップショットに存在しなかった行のPK(先頭カラム)集合。
   * 「新規に増えたKV」のハイライトに使う簡易diff(先頭カラムをPKとみなす)。 */
  newPKs: Set<unknown>
  setSql: (sql: string) => void
  run: () => Promise<void>
  /** id昇順でn件のダミー行を連続INSERTする。1ページに収まる件数(数十〜百件超)を
   * 手でクリックせずに超えられるようにするための、フロントエンド側の便宜機能
   * (SQL言語自体に複数行INSERTを追加したわけではない)。 */
  seedMany: (n: number) => Promise<void>
  reset: () => Promise<void>
}

function collectPKs(tree: TreeSnapshot | null): Set<unknown> {
  const pks = new Set<unknown>()
  if (!tree) return pks
  for (const page of Object.values(tree.pages)) {
    if (!page.isLeaf || !page.rows) continue
    for (const row of page.rows) pks.add(row[0])
  }
  return pks
}

function diffNewPKs(prevTree: TreeSnapshot | null, nextTree: TreeSnapshot | null): Set<unknown> {
  const prevPKs = collectPKs(prevTree)
  const nextPKs = collectPKs(nextTree)
  const diff = new Set<unknown>()
  for (const pk of nextPKs) {
    if (!prevPKs.has(pk)) diff.add(pk)
  }
  return diff
}

function maxNumericId(tree: TreeSnapshot | null): number {
  let max = 0
  if (!tree) return max
  for (const page of Object.values(tree.pages)) {
    if (!page.isLeaf || !page.rows) continue
    for (const row of page.rows) {
      const id = Number(row[0])
      if (!Number.isNaN(id) && id > max) max = id
    }
  }
  return max
}

export const useDbInternal = create<State>((set, get) => ({
  sql: DEFAULT_SQL,
  table: DEFAULT_TABLE,
  busy: false,
  error: null,
  lastResult: null,
  tree: null,
  newPKs: new Set(),

  setSql: (sql) => set({ sql }),

  run: async () => {
    const { sql, table, tree: prevTree } = get()
    set({ busy: true, error: null })
    try {
      const result = await execSql(sql, table)
      if (result.error) {
        set({ busy: false, error: result.error })
        return
      }
      set({
        busy: false,
        lastResult: result,
        tree: result.tree ?? prevTree,
        newPKs: diffNewPKs(prevTree, result.tree ?? null),
      })
    } catch (e) {
      set({ busy: false, error: e instanceof Error ? e.message : String(e) })
    }
  },

  seedMany: async (n) => {
    const { table, tree: prevTree } = get()
    set({ busy: true, error: null })
    try {
      const startId = maxNumericId(prevTree)
      let latestTree: TreeSnapshot | null = prevTree
      let latestResult: ExecResponse | null = null
      for (let i = 1; i <= n; i++) {
        const id = startId + i
        const result = await execSql(`INSERT INTO ${table} VALUES (${id}, 'seed-${id}')`, table)
        latestResult = result
        if (result.error) break
        if (result.tree) latestTree = result.tree
      }
      set({
        busy: false,
        error: latestResult?.error ?? null,
        lastResult: latestResult,
        tree: latestTree,
        newPKs: diffNewPKs(prevTree, latestTree),
      })
    } catch (e) {
      set({ busy: false, error: e instanceof Error ? e.message : String(e) })
    }
  },

  reset: async () => {
    set({ busy: true, error: null })
    try {
      await resetSession()
      set({ busy: false, lastResult: null, tree: null, newPKs: new Set(), sql: DEFAULT_SQL })
    } catch (e) {
      set({ busy: false, error: e instanceof Error ? e.message : String(e) })
    }
  },
}))
