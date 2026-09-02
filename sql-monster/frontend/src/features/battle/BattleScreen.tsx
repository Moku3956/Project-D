import { useGame } from '../../shared/store'
import { SettingsButton } from '../../shared/SettingsButton'
import { MonsterPanel } from './MonsterPanel'
import { PhaseStepper } from './PhaseStepper'
import { SyntaxMap } from './SyntaxMap'
import { WorkspacePanel } from './WorkspacePanel'

function Outcome() {
  const battle = useGame((s) => s.battle)!
  const restart = useGame((s) => s.restart)
  const backToHome = useGame((s) => s.backToHome)

  return (
    <div className="fixed inset-0 z-30 flex items-center justify-center bg-black/75 p-4">
      <div className="w-[360px] rounded-2xl border border-line bg-panel p-6 text-center">
        <h2
          className={`font-display text-3xl font-extrabold ${
            battle.won ? 'text-done' : 'text-danger'
          }`}
        >
          {battle.won ? 'VICTORY' : 'DEFEAT'}
        </h2>
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
            className="flex-1 rounded-lg bg-neon py-3 font-display text-sm font-bold text-deep hover:brightness-110"
          >
            HOME
          </button>
        </div>
      </div>
    </div>
  )
}

/**
 * バトル画面。header-barは廃止済み(docs/battle_screen_ui.md「v2: スクロールしないレイアウト」)。
 * フェーズタブ+設定ボタンの行 → 左60%(モンスター)/右40%(作業スペース) → SQL対応表、の3段構成。
 */
export function BattleScreen() {
  const battle = useGame((s) => s.battle)
  const error = useGame((s) => s.error)

  if (!battle) return null

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 p-4">
      <div className="flex shrink-0 items-center gap-3">
        <PhaseStepper phase={battle.phase} />
        <SettingsButton />
      </div>

      <div className="flex min-h-0 flex-1 gap-4">
        <MonsterPanel battle={battle} />
        <WorkspacePanel battle={battle} />
      </div>

      <SyntaxMap />

      {error && (
        <p className="shrink-0 rounded border border-danger/50 bg-danger/10 px-3 py-2 text-xs text-danger">
          {error}
        </p>
      )}

      {battle.over && <Outcome />}
    </div>
  )
}
