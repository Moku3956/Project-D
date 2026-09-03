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
  error?: string
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
  nextLeafId?: number
}
