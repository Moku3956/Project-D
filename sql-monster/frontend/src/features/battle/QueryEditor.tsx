import { useEffect, useState } from 'react'
import { useGame } from '../../shared/store'
import { PHASE_ACTIONS, isAnalysisPhase, type PhaseNumber } from '../../shared/types'

/** フェーズごとの雛形SQL。プレイヤーはここから書き換えて使う。 */
function template(phase: PhaseNumber, monsterId: number, monsterHP: number): string {
  switch (phase) {
    case 1:
      return `SELECT * FROM monster_weaknesses\nWHERE monster_id = ${monsterId}`
    case 2:
      // 算術式(hp - 50)が言語未対応のため、絶対値で書く必要がある
      // (project_issues.md「算術演算子が言語に存在しない」)
      return `UPDATE monsters SET hp = ${Math.max(0, monsterHP - 50)}\nWHERE id = ${monsterId}`
    case 3:
      return `SELECT * FROM monster_attacks\nWHERE monster_id = ${monsterId}`
    case 4:
      return `SELECT * FROM monster_attacks\nWHERE monster_id = ${monsterId} AND tag = ''`
  }
}

/** EDITORタブの中身。QUERY_COSTの表示はデザイン変更で廃止済み。 */
export function QueryEditor() {
  const battle = useGame((s) => s.battle)!
  const busy = useGame((s) => s.busy)
  const runQuery = useGame((s) => s.runQuery)
  const advance = useGame((s) => s.advance)
  const lastResult = useGame((s) => s.lastResult)

  const [sql, setSql] = useState('')

  useEffect(() => {
    setSql(template(battle.phase, battle.monster.id, battle.monster_hp))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [battle.phase, battle.monster.id])

  return (
    <div className="flex h-full flex-col">
      <div className="flex shrink-0 items-center justify-between pb-2">
        <span className="text-[13px] text-muted">&gt;_ query_planner.sql</span>
        {lastResult?.scan && (
          <span
            className={`text-[11px] font-bold ${
              lastResult.scan === 'index' ? 'text-done' : 'text-muted'
            }`}
          >
            {lastResult.scan === 'index' ? 'INDEX SCAN — 精密' : `${lastResult.scan.toUpperCase()} SCAN`}
          </span>
        )}
      </div>

      <textarea
        value={sql}
        onChange={(e) => setSql(e.target.value)}
        spellCheck={false}
        className="sql-input min-h-[160px] w-full flex-1 resize-none rounded-lg border border-line bg-deep p-4 text-sm text-neon outline-none focus:border-neon/60"
      />

      <div className="flex shrink-0 justify-end gap-2 pt-3">
        {isAnalysisPhase(battle.phase) && (
          <button
            type="button"
            disabled={busy || battle.over}
            onClick={() => void advance()}
            className="rounded-lg border border-line px-4 py-2.5 font-display text-sm font-bold text-muted hover:text-ink disabled:opacity-50"
          >
            NEXT PHASE
          </button>
        )}
        <button
          type="button"
          disabled={busy || battle.over}
          onClick={() => void runQuery(sql)}
          className="flex-1 rounded-lg bg-neon py-2.5 font-display text-sm font-bold text-deep hover:brightness-110 disabled:opacity-50"
        >
          {PHASE_ACTIONS[battle.phase]}
        </button>
      </div>
    </div>
  )
}
