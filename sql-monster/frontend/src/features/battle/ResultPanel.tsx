import type { QueryResult } from '../../shared/types'

/** RESULTタブの中身。クエリ実行後、自動でこのタブに切り替わる。 */
export function ResultPanel({ phase, result }: { phase: number; result: QueryResult | null }) {
  const columns = result?.columns ?? []
  const rows = result?.rows ?? []

  if (result?.error) {
    return (
      <p className="rounded border border-danger/50 bg-danger/10 p-3 text-xs text-danger">
        {result.error}
      </p>
    )
  }

  if (columns.length === 0) {
    return null
  }

  return (
    <div className="flex h-full flex-col overflow-hidden rounded border border-line">
      <div className="min-h-0 flex-1 overflow-auto">
        <table className="w-full text-[11px]">
          <thead className="sticky top-0">
            <tr className="border-b border-line bg-deep">
              {columns.map((c) => (
                <th key={c} className="px-2 py-2 text-left font-bold whitespace-nowrap text-neon">
                  {c}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, i) => (
              <tr key={i} className="border-b border-line/60 last:border-0">
                {row.map((v, j) => (
                  <td key={j} className="px-2 py-2 whitespace-nowrap text-ink">
                    {v === null ? '[NULL]' : String(v)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="shrink-0 border-t border-line bg-deep/60 px-2 py-2 text-[11px] text-muted">
        {rows.length} row(s) selected.
        {/* 防御フェーズは該当行数がそのままブロック率になるので併記する */}
        {phase === 4 && rows.length > 0 && ` BLOCK_RATE = 1 / ${rows.length}`}
      </p>
    </div>
  )
}
