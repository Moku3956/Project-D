import type { Battle } from '../../shared/types'

function Bar({
  label,
  value,
  max,
  color,
}: {
  label: string
  value: number
  max: number
  color: string
}) {
  const ratio = max === 0 ? 0 : Math.max(0, Math.min(1, value / max))
  return (
    <div>
      <div className="flex items-center justify-between text-[11px]">
        <span style={{ color }}>{label}</span>
        <span className="font-bold" style={{ color }}>
          {value} / {max}
        </span>
      </div>
      <div className="mt-1.5 h-2.5 overflow-hidden rounded bg-base">
        <div
          className="h-full transition-[width] duration-300"
          style={{ width: `${ratio * 100}%`, background: color }}
        />
      </div>
    </div>
  )
}

/** 右パネル。プレイヤーのHP・ターン・2種のリソース・ログ。 */
export function PlayerPanel({ battle }: { battle: Battle }) {
  const { resources } = battle

  return (
    <aside className="flex w-[380px] shrink-0 flex-col gap-4 rounded-xl border border-line bg-panel p-4">
      <h3 className="font-display text-base font-bold text-amber">&gt;_ PLAYER</h3>

      <div className="flex items-start justify-between rounded-lg border border-line p-3">
        <div>
          <p className="text-[11px] text-muted">HP</p>
          <p className="font-display text-xl font-extrabold text-done">
            {battle.player_hp} / {battle.player_max_hp}
          </p>
        </div>
        <div className="text-right">
          <p className="text-[11px] text-muted">TURN</p>
          <p className="font-display text-xl font-extrabold text-neon">
            {String(battle.turn).padStart(2, '0')}
          </p>
        </div>
      </div>

      <Bar
        label="ANALYSIS_AP (SELECT)"
        value={resources.analysis}
        max={resources.analysis_max}
        color="var(--color-neon)"
      />
      <Bar
        label="CRUD_ATTACK_AP (UPDATE/INSERT)"
        value={resources.attack_defense}
        max={resources.attack_defense_max}
        color="var(--color-amber)"
      />

      <div className="flex min-h-0 flex-1 flex-col">
        <p className="text-[11px] font-bold text-muted">LOG</p>
        <div className="mt-2 flex-1 overflow-y-auto rounded border border-line bg-base/60 p-3">
          {battle.logs
            .slice()
            .reverse()
            .map((line, i) => (
              <p key={i} className="mb-2 text-[11px] leading-relaxed text-muted last:mb-0">
                {line}
              </p>
            ))}
        </div>
      </div>
    </aside>
  )
}
