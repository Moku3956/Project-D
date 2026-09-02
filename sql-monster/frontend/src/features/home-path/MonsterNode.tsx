import type { Monster } from '../../shared/types'
import { tierColor } from '../../shared/types'

type Props = {
  monster: Monster
  point: { x: number; y: number }
  canvas: { w: number; h: number }
  onSelect: () => void
}

/** 状態ごとの見た目。文言は付けず、色・サイズ・グローだけで表す(docs/home_screen_ui.md)。 */
const SIZE = { cleared: 72, current: 96, locked: 72 }
const OPACITY = { cleared: 0.75, current: 1, locked: 0.6 }

export function MonsterNode({ monster, point, canvas, onSelect }: Props) {
  const color = tierColor(monster.level)
  const size = monster.is_boss ? 126 : SIZE[monster.state]
  const isCurrent = monster.state === 'current'
  const locked = monster.state === 'locked'

  // パスのSVGと同じ座標系に載せるため、キャンバスに対する割合で配置する
  const style: React.CSSProperties = {
    left: `${(point.x / canvas.w) * 100}%`,
    top: `${(point.y / canvas.h) * 100}%`,
    width: size,
    height: size,
    transform: 'translate(-50%, -50%)',
    background: color,
    opacity: OPACITY[monster.state],
    border: isCurrent ? '2px solid #fff' : `2px solid ${color}`,
    boxShadow: isCurrent ? `0 0 36px ${color}` : 'none',
    // ボスだけ六角形にして、ロック中でもシルエットで別格だと分かるようにする
    clipPath: monster.is_boss
      ? 'polygon(50% 0%, 93% 25%, 93% 75%, 50% 100%, 7% 75%, 7% 25%)'
      : undefined,
    borderRadius: monster.is_boss ? undefined : '9999px',
  }

  return (
    <>
      <button
        type="button"
        style={style}
        onClick={onSelect}
        disabled={locked}
        title={locked ? 'まだ挑戦できません' : monster.name}
        aria-label={monster.name}
        className={`absolute ${locked ? 'cursor-not-allowed' : 'cursor-pointer hover:brightness-110'}`}
      />
      {isCurrent && (
        <button
          type="button"
          onClick={onSelect}
          style={{
            left: `${(point.x / canvas.w) * 100}%`,
            top: `calc(${(point.y / canvas.h) * 100}% + ${size / 2 + 18}px)`,
            transform: 'translate(-50%, 0)',
            background: color,
          }}
          className="absolute rounded-lg px-6 py-2 font-display text-sm font-bold text-deep hover:brightness-110"
        >
          START
        </button>
      )}
    </>
  )
}
