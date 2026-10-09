/** Goの db-internal-app/internal/api.tableInfoJSON と対応する。 */
export type TableInfo = {
  name: string
  columns: string[]
}

/** Goの db-internal-app/internal/api.execResponse と対応する。 */
export type ExecResponse = {
  columns?: string[]
  rows?: unknown[][]
  affectedRows: number
  tree?: TreeSnapshot
  wal?: WalRecord[]
  error?: string
}

/** Goの db-internal-app/internal/api.walRecordJSON と対応する。
 * op: "INSERT" | "UPDATE" | "DELETE" | "COMMIT" | "ABORT"。
 * changeKind: "added" | "removed" | "changed"、特定できなかった場合は未設定。 */
export type WalRecord = {
  lsn: number
  txnId: number
  pageId: number
  op: string
  table?: string
  changeKind?: string
  row?: unknown[]
}

/** Goの db-internal-app/internal/api.treeSnapshotJSON と対応する。 */
export type TreeSnapshot = {
  rootPageId: number
  pages: Record<string, PageSnapshot>
}

/** Goの db-internal-app/internal/api.pageSnapshotJSON と対応する。 */
export type PageSnapshot = {
  pageId: number
  isLeaf: boolean
  keys?: string[]
  childPageIds?: number[]
  rightmostChild?: number
  rows?: unknown[][]
  /** rows[i]が属するテーブル名(rowsと同じ長さ・同じ並び順)。1つの物理B+Treeを
   * 全テーブルで共有しているため、1ページに複数テーブルの行が混在しうる。 */
  rowTables?: string[]
  nextLeafId?: number
}
