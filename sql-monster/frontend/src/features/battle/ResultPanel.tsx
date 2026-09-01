import type { QueryResult } from '../../shared/types'

const PHASE_HINTS: Record<number, string> = {
  1: 'monster_weaknesses を SELECT して弱点を探る。読んだ行数だけ ANALYSIS_AP を消費する。',
  2: 'UPDATE / INSERT / DELETE でモンスターを攻撃する。HP差分を実測し、残量を超えるとROLLBACKされる。',
  3: 'monster_attacks を SELECT して、次に来る攻撃を予測する。',
  4: 'WHERE で絞り込んで防御する。ブロック率は 1 / 該当行数。',
}

/**
 * 左パネル。SELECTの結果テーブルを出す。
 * フェーズ2(攻撃)はSELECT結果がないので、代わりに実測ダメージの見方を出す。
 */
export function ResultPanel({ phase, result }: { phase: number; result: QueryResult | null }) {
  const columns = result?.columns ?? []
  const rows = result?.rows ?? []
  const title = phase === 2 ? 'PREVIEW' : 'RESULT'

  return (
    <aside className="flex w-[346px] shrink-0 flex-col justify-end rounded-xl border border-line bg-panel p-4">
      <h3 className="font-display text-base font-bold text-neon">&gt;_ {title}</h3>
      <p className="mt-3 text-xs leading-relaxed text-muted">{PHASE_HINTS[phase]}</p>

      {result?.error && (
        <p className="mt-3 rounded border border-danger/50 bg-danger/10 p-2 text-xs text-danger">
          {result.error}
        </p>
      )}

      {columns.length > 0 && (
        <div className="mt-4 overflow-hidden rounded border border-line">
          <table className="w-full text-[11px]">
            <thead>
              <tr className="border-b border-line bg-base/60">
                {columns.map((c) => (
                  <th key={c} className="px-2 py-2 text-left font-bold text-neon">
                    {c}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, i) => (
                <tr key={i} className="border-b border-line/60 last:border-0">
                  {row.map((v, j) => (
                    <td key={j} className="px-2 py-2 text-ink">
                      {v === null ? '[NULL]' : String(v)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
          <p className="bg-base/60 px-2 py-2 text-[11px] text-muted">
            {rows.length} row(s) selected.
            {/* 防御フェーズは該当行数がそのままブロック率になるので併記する */}
            {phase === 4 && rows.length > 0 && ` BLOCK_RATE = 1 / ${rows.length}`}
          </p>
        </div>
      )}
    </aside>
  )
}
