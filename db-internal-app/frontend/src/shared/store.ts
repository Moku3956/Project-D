import { create } from 'zustand'
import { execSql, listTables, resetSession } from './api'
import type { ExecResponse, TableInfo, TreeSnapshot } from './types'

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

type State = {
  sql: string
  /** 現在選択中(表示対象)のテーブル名。エディターのSQLはこのテーブル宛とは
   * 限らない(CREATE TABLE等で別名を指定できる)が、実行後にB+Treeをダンプ
   * するのは常にこのテーブル。 */
  currentTable: string
  /** セッション内に存在する全テーブル(タブ切り替えUI用)。名前の辞書順。 */
  tables: TableInfo[]
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
  /** タブをクリックしたときに呼ぶ。選択テーブルを切り替えて、そのテーブルの
   * 現在のデータ・B+Treeを取り直す。 */
  switchTable: (name: string) => Promise<void>
  /** 「+ 新しいテーブル」ボタン用。t1, t2, ...という連番の名前でCREATE TABLEを
   * 即実行し、そのテーブルに切り替える(エディターに入力するだけで実行は
   * ユーザー任せ、という以前の挙動はユーザー指示によりやめた)。 */
  createTable: () => Promise<void>
  reset: () => Promise<void>
}

// 1つの物理B+Treeを全テーブルで共有しているため、tree.pages[].rowsには他
// テーブルの行も混ざる。tableに一致する行だけを対象にする(rowTablesで判定)。
function collectPKs(tree: TreeSnapshot | null, table: string): Set<unknown> {
  const pks = new Set<unknown>()
  if (!tree) return pks
  for (const page of Object.values(tree.pages)) {
    if (!page.isLeaf || !page.rows) continue
    page.rows.forEach((row, i) => {
      if ((page.rowTables?.[i] ?? table) === table) pks.add(row[0])
    })
  }
  return pks
}

function diffNewPKs(prevTree: TreeSnapshot | null, nextTree: TreeSnapshot | null, table: string): Set<unknown> {
  const prevPKs = collectPKs(prevTree, table)
  const nextPKs = collectPKs(nextTree, table)
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
  return `dummy-${n}`
}

/** 既存の行のPK(先頭6桁の数値部分)から次に使う番号を決める。手入力された
 * 短いPK("1"等)が混ざっていても、先頭を数値として読める範囲でmaxを取る。
 * (他テーブルの行は無視する。tableに一致する行だけが対象。) */
function maxNumericId(tree: TreeSnapshot | null, table: string): number {
  let max = 0
  if (!tree) return max
  for (const page of Object.values(tree.pages)) {
    if (!page.isLeaf || !page.rows) continue
    page.rows.forEach((row, i) => {
      if ((page.rowTables?.[i] ?? table) !== table) return
      const numeric = String(row[0]).slice(0, ID_NUMERIC_DIGITS)
      const id = Number(numeric)
      if (!Number.isNaN(id) && id > max) max = id
    })
  }
  return max
}

/** 既存テーブル名から「t1, t2, ...」の次の連番を決める。t\d+という名前の
 * テーブルの中の最大値+1(存在しなければt1から)。 */
function nextAutoTableName(tables: TableInfo[]): string {
  let max = 0
  for (const t of tables) {
    const m = /^t(\d+)$/.exec(t.name)
    if (m) max = Math.max(max, Number(m[1]))
  }
  return `t${max + 1}`
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
  sql: '',
  currentTable: DEFAULT_TABLE,
  tables: [],
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
      const tables = await listTables()
      set({ busy: false, lastResult: result, tree: result.tree ?? null, tables })
    } catch (e) {
      set({ busy: false, error: e instanceof Error ? e.message : String(e) })
    }
  },

  run: async () => {
    const { sql, currentTable, tree: prevTree } = get()
    set({ busy: true, error: null })
    try {
      const result = await execSql(sql, currentTable)
      if (result.error) {
        set({ busy: false, error: result.error })
        return
      }
      // CREATE/DROP TABLE等でテーブル一覧が変わっている可能性があるため、
      // 実行のたびにタブ一覧も取り直す。
      const tables = await listTables()
      set({
        busy: false,
        lastResult: result,
        tree: result.tree ?? prevTree,
        newPKs: diffNewPKs(prevTree, result.tree ?? null, currentTable),
        tables,
      })
    } catch (e) {
      set({ busy: false, error: e instanceof Error ? e.message : String(e) })
    }
  },

  seedMany: async (n) => {
    const { currentTable, tree: prevTree } = get()
    set({ busy: true, error: null })
    const BATCH_SIZE = 25 // 直列だと数千件で数分かかるため、まとめて並行実行する
    try {
      const startId = maxNumericId(prevTree, currentTable)
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
              `INSERT INTO ${currentTable} VALUES ('${paddedSeedId(id)}', '${seedName(id)}')`,
              wantTree ? currentTable : undefined,
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
        newPKs: diffNewPKs(prevTree, finalTree, currentTable),
      })
    } catch (e) {
      set({ busy: false, error: e instanceof Error ? e.message : String(e) })
    }
  },

  switchTable: async (name) => {
    set({ busy: true, error: null, currentTable: name })
    try {
      const result = await execSql(`SELECT * FROM ${name}`, name)
      if (result.error) {
        set({ busy: false, error: result.error, tree: null, newPKs: new Set() })
        return
      }
      set({ busy: false, lastResult: result, tree: result.tree ?? null, newPKs: new Set() })
    } catch (e) {
      set({ busy: false, error: e instanceof Error ? e.message : String(e) })
    }
  },

  createTable: async () => {
    const { tables } = get()
    const name = nextAutoTableName(tables)
    const sql = `CREATE TABLE ${name} (id VARCHAR(${ID_COLUMN_LENGTH}) PRIMARY KEY, name VARCHAR(${NAME_COLUMN_LENGTH}))`
    set({ busy: true, error: null, sql })
    try {
      const result = await execSql(sql, name)
      if (result.error) {
        set({ busy: false, error: result.error })
        return
      }
      const newTables = await listTables()
      set({
        busy: false,
        lastResult: result,
        tree: result.tree ?? null,
        tables: newTables,
        currentTable: name,
        newPKs: new Set(),
      })
    } catch (e) {
      set({ busy: false, error: e instanceof Error ? e.message : String(e) })
    }
  },

  reset: async () => {
    set({ busy: true, error: null })
    try {
      await resetSession()
      set({
        busy: false,
        lastResult: null,
        tree: null,
        newPKs: new Set(),
        sql: '',
        currentTable: DEFAULT_TABLE,
        tables: [],
      })
      await get().init()
    } catch (e) {
      set({ busy: false, error: e instanceof Error ? e.message : String(e) })
    }
  },
}))
