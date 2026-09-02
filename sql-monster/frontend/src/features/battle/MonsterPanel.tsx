import { useState } from 'react'
import type { Battle } from '../../shared/types'
import { tierColor } from '../../shared/types'
import { SchemaDiagram } from './SchemaDiagram'

type Tab = 'card' | 'table'

function CardView({ battle }: { battle: Battle }) {
  const { monster, monster_hp } = battle
  const color = tierColor(monster.level)
  const ratio = Math.max(0, Math.min(1, monster_hp / monster.max_hp))

  return (
    <div className="flex h-full flex-col">
      <div
        className="min-h-0 flex-1 rounded-lg border"
        style={{ borderColor: color, background: `${color}14` }}
      >
        <div className="flex h-full items-center justify-center">
          <span className="text-sm text-faint">[ MONSTER ART — TBD ]</span>
        </div>
      </div>

      <div className="mt-4 shrink-0">
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

        <div className="relative mt-3 h-6 overflow-hidden rounded bg-deep">
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
    </div>
  )
}

/** 左カラム。CARD(モンスター表示)とTABLE(ER図)をタブで切り替える。 */
export function MonsterPanel({ battle }: { battle: Battle }) {
  const [tab, setTab] = useState<Tab>('card')

  return (
    <section className="flex min-h-0 flex-[3] flex-col gap-2">
      <div className="flex shrink-0 gap-2">
        {(['card', 'table'] as const).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(t)}
            className={`rounded-md border px-4 py-2 font-display text-[11px] font-bold uppercase ${
              tab === t ? 'border-neon bg-neon/15 text-neon' : 'border-line bg-deep text-muted'
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      <div className="min-h-0 flex-1 rounded-xl border border-line bg-panel p-4">
        {tab === 'card' ? <CardView battle={battle} /> : <SchemaDiagram />}
      </div>
    </section>
  )
}
