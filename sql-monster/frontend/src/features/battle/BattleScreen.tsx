import { useGame } from '../../shared/store'
import { MonsterDisplay } from './MonsterDisplay'
import { PhaseStepper } from './PhaseStepper'
import { PlayerPanel } from './PlayerPanel'
import { QueryEditor } from './QueryEditor'
import { ResultPanel } from './ResultPanel'
import { SyntaxMap } from './SyntaxMap'

/** 決着がついたときに前面に出す結果表示。 */
function Outcome() {
  const battle = useGame((s) => s.battle)!
  const restart = useGame((s) => s.restart)
  const backToHome = useGame((s) => s.backToHome)

  return (
    <div className="fixed inset-0 z-30 flex items-center justify-center bg-black/75 p-4">
      <div className="w-[420px] rounded-2xl border border-line bg-panel p-6 text-center">
        <h2
          className={`font-display text-3xl font-extrabold ${
            battle.won ? 'text-done' : 'text-danger'
          }`}
        >
          {battle.won ? 'VICTORY' : 'DEFEAT'}
        </h2>
        <p className="mt-3 text-sm text-muted">
          {battle.won
            ? `${battle.monster.name} を撃破しました。次のモンスターが解放されます。`
            : `${battle.monster.name} に敗北しました。`}
        </p>
        <div className="mt-6 flex gap-3">
          <button
            type="button"
            onClick={() => void restart()}
            className="flex-1 rounded-lg border border-line py-3 font-display text-sm font-bold text-muted hover:text-ink"
          >
            RESTART
          </button>
          <button
            type="button"
            onClick={() => void backToHome()}
            className="flex-1 rounded-lg bg-neon py-3 font-display text-sm font-bold text-base hover:brightness-110"
          >
            HOME
          </button>
        </div>
      </div>
    </div>
  )
}

export function BattleScreen() {
  const battle = useGame((s) => s.battle)
  const lastResult = useGame((s) => s.lastResult)
  const error = useGame((s) => s.error)

  if (!battle) return null

  return (
    <div className="flex flex-col gap-4 pb-6">
      <PhaseStepper phase={battle.phase} />
      <MonsterDisplay battle={battle} />

      <div className="flex gap-4 px-6">
        <ResultPanel phase={battle.phase} result={lastResult} />
        <QueryEditor />
        <PlayerPanel battle={battle} />
      </div>

      <SyntaxMap />

      {error && (
        <p className="mx-6 rounded border border-danger/50 bg-danger/10 p-3 text-xs text-danger">
          {error}
        </p>
      )}

      {battle.over && <Outcome />}
    </div>
  )
}
