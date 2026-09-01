import { PHASE_LABELS, type PhaseNumber } from '../../shared/types'

/**
 * 画面最上部の全幅ステッパー。状態は色だけで表す(DONE=緑 / 進行中=シアン / 未着手=グレー)。
 * 文言は付けない — docs/battle_screen_ui.md「turn-phasesの配置変更」参照。
 */
export function PhaseStepper({ phase }: { phase: PhaseNumber }) {
  const phases: PhaseNumber[] = [1, 2, 3, 4]

  return (
    <section className="mx-6 rounded-xl border border-line bg-panel p-3">
      <p className="px-1 pb-2 text-[11px] font-bold text-muted">BATTLE_FLOW_PHASE</p>
      <div className="grid grid-cols-4 gap-2">
        {phases.map((p) => {
          const done = p < phase
          const active = p === phase
          return (
            <div
              key={p}
              className={`rounded-lg border px-3 py-3 ${
                active ? 'border-neon bg-neon/15' : 'border-line bg-base/40'
              }`}
            >
              <p
                className={`text-[11px] font-bold ${
                  active ? 'text-neon' : done ? 'text-done' : 'text-muted'
                }`}
              >
                PHASE_0{p}
              </p>
              <p
                className={`font-display text-sm font-bold ${active ? 'text-ink' : 'text-muted'}`}
              >
                {PHASE_LABELS[p]}
              </p>
            </div>
          )
        })}
      </div>
    </section>
  )
}
