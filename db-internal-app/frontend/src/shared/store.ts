import { create } from 'zustand'
import { execSql, listTables, resetSession } from './api'
import { buildTemplate, type SqlMode } from './sqlTemplates'
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
  /** ランダムなid・名前でn件の行を連続INSERTする。1ページに収まる件数
   * (実測数件)を手でクリックせずに超えられるようにするための、フロントエンド
   * 側の便宜機能(SQL言語自体に複数行INSERTを追加したわけではない)。
   * 「Add Random」(n=1)と「Bulk Insert」(n=任意件数)の両方がこれを呼ぶ。 */
  seedMany: (n: number) => Promise<void>
  /** タブをクリックしたときに呼ぶ。選択テーブルを切り替えて、そのテーブルの
   * 現在のデータ・B+Treeを取り直す。 */
  switchTable: (name: string) => Promise<void>
  /** 「+ 新しいテーブル」ボタン用。t1, t2, ...という連番の名前でCREATE TABLEを
   * 即実行し、そのテーブルに切り替える(エディターに入力するだけで実行は
   * ユーザー任せ、という以前の挙動はユーザー指示によりやめた)。 */
  createTable: () => Promise<void>
  /** エディターのINSERT/UPDATE/DELETE切り替えボタン用。現在の状態。 */
  sqlMode: SqlMode
  /** モードを切り替えつつ、選択中テーブルのカラムに沿ったテンプレートSQL
   * (値は空クオート)をエディターに差し込む。 */
  applySqlMode: (mode: SqlMode) => void
  /** テーブルの行をクリックしたときに呼ぶ。「Add Random」「まとめて追加」で
   * 作った行はB+Tree分岐用にPKがパディングされた長い文字列になっており、
   * 表示上の短い値(例: "1")をWHERE句に手入力しても実際のPKと一致せず
   * ヒットしない(UPDATE/DELETEが「動かない」という実害があった)。行クリックで
   * その行の実際のPK値をWHERE句にそのまま埋め込むことで確実にヒットさせる。
   * INSERTモード中にクリックした場合は、既存行を触る操作だと考えUPDATEに
   * 切り替える。 */
  fillTemplateForRow: (pkValue: string) => void
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

// 「dummy-*」ではなく実在の名前らしい値を入れたい、というユーザー指示による。
// ダミーだと分かる無機質な値より、実データっぽい方がB+Treeの中身として自然に見える。
const REALISTIC_NAMES = [
  'Alice', 'Bob', 'Charlie', 'Diana', 'Ethan', 'Fiona', 'George', 'Hannah',
  'Ivan', 'Julia', 'Kevin', 'Laura', 'Mike', 'Nina', 'Oscar', 'Paula',
  'Quinn', 'Rachel', 'Sam', 'Tina', 'Uma', 'Victor', 'Wendy', 'Xander',
  'Yara', 'Zoe',
]

function randomName(): string {
  return REALISTIC_NAMES[Math.floor(Math.random() * REALISTIC_NAMES.length)]
}

/** 既存の行のPK(先頭6桁の数値部分)を全て集める。手入力された短いPK
 * ("1"等)が混ざっていても、先頭を数値として読める範囲で拾う。(他テーブルの
 * 行は無視する。tableに一致する行だけが対象。)ランダムなidを採番する際の
 * 重複チェックに使う。 */
function existingNumericIds(tree: TreeSnapshot | null, table: string): Set<number> {
  const ids = new Set<number>()
  if (!tree) return ids
  for (const page of Object.values(tree.pages)) {
    if (!page.isLeaf || !page.rows) continue
    page.rows.forEach((row, i) => {
      if ((page.rowTables?.[i] ?? table) !== table) return
      const numeric = String(row[0]).slice(0, ID_NUMERIC_DIGITS)
      const id = Number(numeric)
      if (!Number.isNaN(id)) ids.add(id)
    })
  }
  return ids
}

// ID_NUMERIC_DIGITSが6桁なのでこの範囲に収める。
const MAX_RANDOM_ID = 999999

/** 「連番じゃなくてランダムにしよう」というユーザー指示による。既存の行(他
 * テーブルの行を除く)・同一バッチ内のidと重複しない範囲で、n件のランダムな
 * 数値idを選ぶ。 */
function randomUniqueIds(n: number, exclude: Set<number>): number[] {
  const used = new Set(exclude)
  const picked: number[] = []
  while (picked.length < n) {
    const candidate = 1 + Math.floor(Math.random() * MAX_RANDOM_ID)
    if (used.has(candidate)) continue
    used.add(candidate)
    picked.push(candidate)
  }
  return picked
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
  sqlMode: 'INSERT',

  setSql: (sql) => set({ sql }),

  applySqlMode: (mode) => {
    const { currentTable, tables } = get()
    const columns = tables.find((t) => t.name === currentTable)?.columns ?? []
    set({ sqlMode: mode, sql: buildTemplate(mode, currentTable, columns) })
  },

  fillTemplateForRow: (pkValue) => {
    const { currentTable, tables, sqlMode } = get()
    const columns = tables.find((t) => t.name === currentTable)?.columns ?? []
    const mode = sqlMode === 'INSERT' ? 'UPDATE' : sqlMode
    set({ sqlMode: mode, sql: buildTemplate(mode, currentTable, columns, pkValue) })
  },

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
      const ids = randomUniqueIds(n, existingNumericIds(prevTree, currentTable))
      let firstError: string | null = null
      let finalResult: ExecResponse | null = null
      // 木のダンプはサーバー側でページ全体を辿るコストがあるため、毎回は要求せず
      // 一番最後のINSERTでだけツリーを取得する。
      for (let batchStart = 0; batchStart < n && !firstError; batchStart += BATCH_SIZE) {
        const batchEnd = Math.min(batchStart + BATCH_SIZE, n)
        const results = await Promise.all(
          ids.slice(batchStart, batchEnd).map((id, k) => {
            const wantTree = batchStart + k === n - 1
            return execSql(
              `INSERT INTO ${currentTable} VALUES ('${paddedSeedId(id)}', '${randomName()}')`,
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
    // 実行するCREATE TABLE文はエディターには表示しない(自動生成された裏側の
    // クエリなので、ユーザー指示によりエディターの内容は変えない)。
    const createSql = `CREATE TABLE ${name} (id VARCHAR(${ID_COLUMN_LENGTH}) PRIMARY KEY, name VARCHAR(${NAME_COLUMN_LENGTH}))`
    set({ busy: true, error: null })
    try {
      const result = await execSql(createSql, name)
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
        sqlMode: 'INSERT',
      })
      await get().init()
    } catch (e) {
      set({ busy: false, error: e instanceof Error ? e.message : String(e) })
    }
  },
}))
