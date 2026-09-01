import type { Battle } from '../../shared/types'
import { tierColor } from '../../shared/types'

/** モンスターのビジュアルとHPバー。弱点はここには出さない(SQLで見つけるもの)。 */
export function MonsterDisplay({ battle }: { battle: Battle }) {
  const { monster, monster_hp } = battle
  const color = tierColor(monster.level)
  const ratio = Math.max(0, Math.min(1, monster_hp / monster.max_hp))

  return (
    <section className="mx-6 rounded-xl border border-line bg-panel p-6">
      <div
        className="mx-auto flex h-[220px] max-w-[800px] items-center justify-center rounded-lg border"
        style={{ borderColor: color, background: `${color}14` }}
      >
        <span className="text-sm text-faint">[ MONSTER ART — TBD ]</span>
      </div>

      <div className="mx-auto mt-5 max-w-[800px]">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <h2 className="font-display text-2xl font-extrabold text-ink">{monster.name}</h2>
            {monster.is_boss && (
              <span className="rounded border border-danger px-2 py-0.5 text-[11px] font-bold text-danger">
                BOSS
              </span>
            )}
          </div>
          <span className="font-display text-lg font-bold" style={{ color }}>
            LVL {monster.level}
          </span>
        </div>

        <div className="relative mt-3 h-6 overflow-hidden rounded bg-base">
          <div
            className="h-full transition-[width] duration-300"
            style={{ width: `${ratio * 100}%`, background: color }}
          />
          <span className="absolute inset-0 flex items-center justify-center text-[13px] text-ink">
            HP: {monster_hp.toLocaleString()} / {monster.max_hp.toLocaleString()} (
            {Math.round(ratio * 100)}%)
          </span>
        </div>
      </div>
    </section>
  )
}
