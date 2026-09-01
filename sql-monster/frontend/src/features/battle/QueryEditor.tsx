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

export function QueryEditor() {
  const battle = useGame((s) => s.battle)!
  const busy = useGame((s) => s.busy)
  const runQuery = useGame((s) => s.runQuery)
  const advance = useGame((s) => s.advance)
  const lastResult = useGame((s) => s.lastResult)

  const [sql, setSql] = useState('')

  // フェーズが変わったら雛形を入れ直す
  useEffect(() => {
    setSql(template(battle.phase, battle.monster.id, battle.monster_hp))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [battle.phase, battle.monster.id])

  const cost =
    battle.phase === 2
      ? 'PROJECTED_COST: 実測したHP差分ぶん (ATTACK/DEFENSE)'
      : `QUERY_COST: 読んだ行数ぶん (ANALYSIS)`

  return (
    <section className="flex min-w-0 flex-1 flex-col gap-4">
      <div className="rounded-xl border border-line bg-panel p-4">
        <div className="flex items-center justify-between pb-3">
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
          className="sql-input h-[200px] w-full resize-none rounded-lg border border-line bg-base p-4 text-sm text-neon outline-none focus:border-neon/60"
        />

        <div className="flex items-center justify-between pt-4">
          <span className="text-[13px] text-muted">{cost}</span>
          <div className="flex gap-2">
            {isAnalysisPhase(battle.phase) && (
              <button
                type="button"
                disabled={busy || battle.over}
                onClick={() => void advance()}
                className="rounded-lg border border-line px-4 py-3 font-display text-sm font-bold text-muted hover:text-ink disabled:opacity-50"
              >
                NEXT PHASE
              </button>
            )}
            <button
              type="button"
              disabled={busy || battle.over}
              onClick={() => void runQuery(sql)}
              className="rounded-lg bg-neon px-6 py-3 font-display text-sm font-bold text-base hover:brightness-110 disabled:opacity-50"
            >
              {PHASE_ACTIONS[battle.phase]}
            </button>
          </div>
        </div>
      </div>
    </section>
  )
}
