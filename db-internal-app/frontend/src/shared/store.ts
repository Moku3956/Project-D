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

// 1ページ(4KB)に3〜4件しか入らないよう、わざと長いnameで1件あたりのバイト数を
// 稼ぐ(実データの話。表示側は features/storage/layout.ts の truncateField で
// 常に短く見せる)。VARCHAR(50)という宣言は今のexecutor/plannerでは検証されて
// いない(project_issuesの既知の課題)ため、実際は好きな長さを入れられる。
const PADDED_NAME_LENGTH = 700

function paddedSeedName(id: number): string {
  return `seed-${id}`.padEnd(PADDED_NAME_LENGTH, 'x')
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
    const BATCH_SIZE = 25 // 直列だと数千件で数分かかるため、まとめて並行実行する
    try {
      const startId = maxNumericId(prevTree)
      let firstError: string | null = null
      let finalResult: ExecResponse | null = null
      // 木のダンプはサーバー側でページ全体を辿るコストがあるため、毎回は要求せず
      // 一番最後のINSERTでだけツリーを取得する。
      for (let batchStart = 1; batchStart <= n && !firstError; batchStart += BATCH_SIZE) {
        const batchEnd = Math.min(batchStart + BATCH_SIZE - 1, n)
        const results = await Promise.all(
          Array.from({ length: batchEnd - batchStart + 1 }, (_, k) => {
            const id = startId + batchStart + k
            const wantTree = batchStart + k === n
            return execSql(`INSERT INTO ${table} VALUES (${id}, '${paddedSeedName(id)}')`, wantTree ? table : undefined)
          }),
        )
        const errored = results.find((r) => r.error)
        if (errored) firstError = errored.error ?? '不明なエラー'
        const withTree = results.find((r) => r.tree)
        if (withTree) finalResult = withTree
      }

      const finalTree = finalResult?.tree ?? prevTree
      set({
        busy: false,
        error: firstError,
        lastResult: finalResult,
        tree: finalTree,
        newPKs: diffNewPKs(prevTree, finalTree),
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
