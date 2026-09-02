import { useGame } from '../../shared/store'
import { tierColor } from '../../shared/types'

/**
 * ノードをタップしたときに出るプレビュー。
 * 弱点(WEAKNESS)はここでは見せない。SQLで分析して見つけるものだから(spec.md)。
 */
export function MonsterPreview() {
  const monster = useGame((s) => s.previewing)
  const close = useGame((s) => s.closePreview)
  const startBattle = useGame((s) => s.startBattle)
  const busy = useGame((s) => s.busy)

  if (!monster) return null
  const color = tierColor(monster.level)

  return (
    <div
      className="fixed inset-0 z-30 flex items-center justify-center bg-black/70 p-4"
      onClick={close}
    >
      <div
        className="w-[480px] rounded-2xl border border-line bg-panel p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex justify-end">
          <button type="button" onClick={close} className="text-muted hover:text-ink">
            x
          </button>
        </div>

        {/* モンスターの絵。現状は素材がないのでレベル色のプレースホルダー */}
        <div
          className="mt-4 flex h-[240px] items-center justify-center rounded-lg border"
          style={{ borderColor: color, background: `${color}14` }}
        >
          <span className="text-sm text-faint">[ MONSTER ART — TBD ]</span>
        </div>

        <div className="mt-4 flex items-center justify-between">
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

        <p className="mt-3 text-sm text-muted">
          MAX_HP: <span className="font-bold text-ink">{monster.max_hp.toLocaleString()}</span>
        </p>

        <button
          type="button"
          disabled={busy}
          onClick={() => void startBattle(monster.id)}
          className="mt-4 w-full rounded-lg py-3 font-display text-sm font-bold text-deep hover:brightness-110 disabled:opacity-50"
          style={{ background: color }}
        >
          START BATTLE
        </button>
      </div>
    </div>
  )
}
