import { useEffect } from 'react'
import { useGame } from '../../shared/store'
import { MonsterNode } from './MonsterNode'

/** ジグザグ配置の座標(path-sectionのローカル座標系)。Figmaの配置に合わせてある。 */
const LAYOUT = [
  { x: 220, y: 80 },
  { x: 696, y: 230 },
  { x: 1172, y: 380 },
  { x: 696, y: 540 },
  { x: 220, y: 690 },
  { x: 696, y: 850 },
]

const CANVAS = { w: 1392, h: 950 }

/**
 * ホーム画面。モンスターを1本道のパス上に並べ、クリア済みまでを実線、
 * 未開放の先を点線でつなぐ(docs/home_screen_ui.md)。
 */
export function HomePath() {
  const monsters = useGame((s) => s.monsters)
  const loadMonsters = useGame((s) => s.loadMonsters)
  const openPreview = useGame((s) => s.openPreview)

  useEffect(() => {
    void loadMonsters()
  }, [loadMonsters])

  const points = monsters.map((_, i) => LAYOUT[i] ?? LAYOUT[LAYOUT.length - 1])
  // 実線で結ぶのは「クリア済み〜現在地」まで。そこから先は未開放なので点線にする
  const currentIndex = monsters.findIndex((m) => m.state === 'current')
  const solidUntil = currentIndex === -1 ? monsters.length - 1 : currentIndex

  return (
    <div className="p-6">
      <div
        className="relative w-full overflow-hidden rounded-xl border border-line bg-panel"
        style={{ aspectRatio: `${CANVAS.w} / ${CANVAS.h}` }}
      >
        <svg
          viewBox={`0 0 ${CANVAS.w} ${CANVAS.h}`}
          className="absolute inset-0 h-full w-full"
          preserveAspectRatio="none"
        >
          {/* データベースのスキーマ図を思わせる薄いグリッド背景 */}
          <defs>
            <pattern id="grid" width="48" height="48" patternUnits="userSpaceOnUse">
              <path
                d="M 48 0 L 0 0 0 48"
                fill="none"
                stroke="var(--color-neon)"
                strokeOpacity="0.06"
                strokeWidth="1"
              />
            </pattern>
          </defs>
          <rect width={CANVAS.w} height={CANVAS.h} fill="url(#grid)" />

          {points.slice(0, -1).map((p, i) => {
            const next = points[i + 1]
            const solid = i < solidUntil
            return (
              <line
                key={i}
                x1={p.x}
                y1={p.y}
                x2={next.x}
                y2={next.y}
                stroke={solid ? 'var(--color-neon)' : 'var(--color-line-soft)'}
                strokeWidth={solid ? 3 : 2}
                strokeDasharray={solid ? undefined : '8 8'}
              />
            )
          })}
        </svg>

        {monsters.map((m, i) => (
          <MonsterNode
            key={m.id}
            monster={m}
            point={points[i]}
            canvas={CANVAS}
            onSelect={() => openPreview(m)}
          />
        ))}
      </div>
    </div>
  )
}
