import type { TreeSnapshot } from '../../shared/types'
import { stripPadding } from './displayValue'

/** TreeSnapshotの全葉ページからKVを集めて、普通のテーブル(行・列)として
 * 表示する。B+Treeのページ構造(内部の保存のされ方)と対比して、「見た目は
 * ただのテーブルだが、実際はこう保存されている」というのを伝えるための表示。
 * ページ構造とは違い、省略(先頭2件+省略+末尾2件)はせず全件そのまま出す。 */
function collectRows(tree: TreeSnapshot): unknown[][] {
  const rows: unknown[][] = []
  for (const page of Object.values(tree.pages)) {
    if (!page.isLeaf || !page.rows) continue
    rows.push(...page.rows)
  }
  // usersのidは数値ベースの想定だが、任意テーブルのPKは数値とは限らないため
  // 数値比較できなければ文字列としてソートする。
  rows.sort((a, b) => {
    const na = Number(stripPadding(a[0]))
    const nb = Number(stripPadding(b[0]))
    if (!Number.isNaN(na) && !Number.isNaN(nb)) return na - nb
    return String(stripPadding(a[0])).localeCompare(String(stripPadding(b[0])))
  })
  return rows
}

export function DataTable({ tree, columns }: { tree: TreeSnapshot; columns: string[] }) {
  const rows = collectRows(tree)

  return (
    <div className="overflow-x-auto rounded-xl border border-line">
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-line bg-bg">
            {columns.map((c) => (
              <th key={c} className="px-4 py-2 font-mono text-xs font-bold text-muted">
                {c}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="px-4 py-3 text-xs text-muted">
                まだ行がありません
              </td>
            </tr>
          ) : (
            rows.map((row, i) => (
              <tr key={i} className="border-b border-line last:border-b-0">
                {row.map((v, j) => (
                  <td key={j} className="px-4 py-2 font-mono text-ink">
                    {stripPadding(v)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}
