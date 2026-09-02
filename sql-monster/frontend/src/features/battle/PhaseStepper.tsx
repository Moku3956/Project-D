import { PHASE_LABELS, type PhaseNumber } from '../../shared/types'

/**
 * フェーズタブ。状態は色だけで表す(進行中=シアン / それ以外=グレー)。
 * ラベルは付けない — docs/battle_screen_ui.md「turn-phasesの配置変更」参照。
 */
export function PhaseStepper({ phase }: { phase: PhaseNumber }) {
  const phases: PhaseNumber[] = [1, 2, 3, 4]

  return (
    <div className="grid flex-1 grid-cols-4 gap-2 rounded-xl border border-line bg-panel p-3">
      {phases.map((p) => {
        const active = p === phase
        return (
          <div
            key={p}
            className={`rounded-lg border px-3 py-3 ${
              active ? 'border-neon bg-neon/15' : 'border-line bg-deep/40'
            }`}
          >
            <p className={`text-[11px] font-bold ${active ? 'text-neon' : 'text-muted'}`}>
              PHASE_0{p}
            </p>
            <p className={`font-display text-sm font-bold ${active ? 'text-ink' : 'text-muted'}`}>
              {PHASE_LABELS[p]}
            </p>
          </div>
        )
      })}
    </div>
  )
}
