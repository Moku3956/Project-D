import { create } from 'zustand'
import { execSql, resetSession } from './api'
import type { ExecResponse, TreeSnapshot } from './types'

const DEFAULT_TABLE = 'users'

// 内部ノードのセルは[複合キー][子ページID 4byte]だけで、name等の他カラムを
// 一切含まない。そのためnameだけを長くパディングしても、内部ノードの容量
// (実測200件近く)は全く変わらず、Rootが極端に横広がりになってしまう
// (実機フィードバックで発覚)。複合キーは葉・内部ノード両方に含まれるため、
// 主キー(id)自体を長くパディングすることで両方の容量を一緒に下げる。
// 文字列は辞書順ソートされる(storage/btree/cell.go compareValues)ため、
// "id-2" < "id-10" のような直感に反する並びを避けるべく、数値部分を
// ゼロ埋めしてから残りをパディングする。
const ID_NUMERIC_DIGITS = 6 // 999,999件まで辞書順=数値順が一致する
const ID_COLUMN_LENGTH = 600
const NAME_COLUMN_LENGTH = 50 // nameはもうパディングしないので実際の値で足りる長さ

const INIT_SQL = `CREATE TABLE ${DEFAULT_TABLE} (id VARCHAR(${ID_COLUMN_LENGTH}) PRIMARY KEY, name VARCHAR(${NAME_COLUMN_LENGTH}))`
const EDITOR_PLACEHOLDER_SQL = `INSERT INTO ${DEFAULT_TABLE} VALUES ('1', 'Alice')`

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
  /** セッション開始時に自動で呼ぶ。デモ用テーブルを自分で作らせるのは
   * ユーザーにとって無駄な手順、というユーザー指示により、初回アクセス
   * (またはリセット後)に自動でCREATE TABLEしておく。 */
  init: () => Promise<void>
  run: () => Promise<void>
  /** id昇順でn件のダミー行を連続INSERTする。1ページに収まる件数(実測数件)を
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

/** ゼロ埋めした数値+パディング文字、という形のPK文字列を作る。
 * 例: paddedSeedId(42) -> "000042xxxx...x"(合計600文字) */
function paddedSeedId(n: number): string {
  const numeric = String(n).padStart(ID_NUMERIC_DIGITS, '0')
  return numeric.padEnd(ID_COLUMN_LENGTH, 'x')
}

function seedName(n: number): string {
  return `seed-${n}`
}

/** 既存の行のPK(先頭6桁の数値部分)から次に使う番号を決める。手入力された
 * 短いPK("1"等)が混ざっていても、先頭を数値として読める範囲でmaxを取る。 */
function maxNumericId(tree: TreeSnapshot | null): number {
  let max = 0
  if (!tree) return max
  for (const page of Object.values(tree.pages)) {
    if (!page.isLeaf || !page.rows) continue
    for (const row of page.rows) {
      const numeric = String(row[0]).slice(0, ID_NUMERIC_DIGITS)
      const id = Number(numeric)
      if (!Number.isNaN(id) && id > max) max = id
    }
  }
  return max
}

/** セッションのDBに(まだなければ)デモ用テーブルを作る。他人と共有するセッション
 * ではない前提だが、既に存在する場合はcatalog.goの"table \"users\" already
 * exists"エラーが返るだけなので、それは無視して現在の木を取得し直す。 */
async function ensureTable(): Promise<ExecResponse> {
  const result = await execSql(INIT_SQL, DEFAULT_TABLE)
  if (!result.error) return result
  if (!result.error.includes('already exists')) return result
  return execSql(`SELECT * FROM ${DEFAULT_TABLE}`, DEFAULT_TABLE)
}

export const useDbInternal = create<State>((set, get) => ({
  sql: EDITOR_PLACEHOLDER_SQL,
  table: DEFAULT_TABLE,
  busy: false,
  error: null,
  lastResult: null,
  tree: null,
  newPKs: new Set(),

  setSql: (sql) => set({ sql }),

  init: async () => {
    set({ busy: true, error: null })
    try {
      const result = await ensureTable()
      if (result.error) {
        set({ busy: false, error: result.error })
        return
      }
      set({ busy: false, lastResult: result, tree: result.tree ?? null })
    } catch (e) {
      set({ busy: false, error: e instanceof Error ? e.message : String(e) })
    }
  },

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
            return execSql(
              `INSERT INTO ${table} VALUES ('${paddedSeedId(id)}', '${seedName(id)}')`,
              wantTree ? table : undefined,
            )
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
      set({ busy: false, lastResult: null, tree: null, newPKs: new Set(), sql: EDITOR_PLACEHOLDER_SQL })
      await get().init()
    } catch (e) {
      set({ busy: false, error: e instanceof Error ? e.message : String(e) })
    }
  },
}))
