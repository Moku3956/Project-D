import type { TreeSnapshot } from '../../shared/types'
import { useI18n } from '../../shared/i18n'
import { useDbInternal } from '../../shared/store'
import { stripPadding } from './displayValue'

/** TreeSnapshotの全葉ページからKVを集めて、普通のテーブル(行・列)として
 * 表示する。B+Treeのページ構造(内部の保存のされ方)と対比して、「見た目は
 * ただのテーブルだが、実際はこう保存されている」というのを伝えるための表示。
 * ページ構造とは違い、省略(先頭2件+省略+末尾2件)はせず全件そのまま出す。
 * 1つの物理B+Treeを全テーブルで共有しているため、page.rowsには他テーブルの
 * 行も混ざる。tableに一致する行だけを対象にする(rowTablesで判定)。 */
function collectRows(tree: TreeSnapshot, table: string): unknown[][] {
  const rows: unknown[][] = []
  for (const page of Object.values(tree.pages)) {
    if (!page.isLeaf || !page.rows) continue
    page.rows.forEach((row, i) => {
      if ((page.rowTables?.[i] ?? table) === table) rows.push(row)
    })
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

export function DataTable({ tree, columns, table }: { tree: TreeSnapshot; columns: string[]; table: string }) {
  const rows = collectRows(tree, table)
  const { t } = useI18n()
  const fillTemplateForRow = useDbInternal((s) => s.fillTemplateForRow)

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
                {t('dataTableEmpty')}
              </td>
            </tr>
          ) : (
            rows.map((row, i) => (
              <tr
                key={i}
                onClick={() => fillTemplateForRow(String(row[0]))}
                title={t('rowClickHint')}
                className="cursor-pointer border-b border-line last:border-b-0 hover:bg-bg"
              >
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
