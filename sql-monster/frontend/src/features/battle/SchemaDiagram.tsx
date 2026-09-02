type Column = readonly [name: string, type: string, key?: 'PK' | 'FK']

type Entity = {
  name: string
  x: number
  y: number
  w: number
  columns: Column[]
  accent: 'hub' | 'related' | 'unrelated'
}

/** 設計上の座標系(Figmaの`er-diagram`フレームと同じ)。実表示はこれを%換算して配置する。 */
const CANVAS = { w: 822, h: 521 }

// sql-monster/internal/game/schema.go のテーブル定義と一致させてある
const ENTITIES: Entity[] = [
  {
    name: 'monsters',
    x: 40,
    y: 90,
    w: 220,
    accent: 'hub',
    columns: [
      ['id', 'INT', 'PK'],
      ['name', 'VARCHAR(50)'],
      ['hp', 'INT'],
      ['weakness', 'VARCHAR(20)'],
    ],
  },
  {
    name: 'monster_weaknesses',
    x: 400,
    y: 30,
    w: 260,
    accent: 'related',
    columns: [
      ['id', 'INT', 'PK'],
      ['monster_id', 'INT', 'FK'],
      ['dmg_type', 'VARCHAR(20)'],
      ['severity', 'INT'],
    ],
  },
  {
    name: 'monster_attacks',
    x: 400,
    y: 230,
    w: 260,
    accent: 'related',
    columns: [
      ['id', 'INT', 'PK'],
      ['monster_id', 'INT', 'FK'],
      ['tag', 'VARCHAR(20)'],
      ['likelihood', 'INT'],
      ['power', 'INT'],
    ],
  },
  {
    name: 'players',
    x: 40,
    y: 430,
    w: 200,
    accent: 'unrelated',
    columns: [
      ['id', 'INT', 'PK'],
      ['hp', 'INT'],
    ],
  },
]

// monsters(右辺中央) → monster_weaknesses / monster_attacks(左辺中央)
const CONNECTORS = [
  { x1: 260, y1: 150, x2: 400, y2: 90 },
  { x1: 260, y1: 150, x2: 400, y2: 299 },
]

const ACCENT = {
  hub: { border: 'border-ink', header: 'text-ink' },
  related: { border: 'border-neon', header: 'text-neon' },
  unrelated: { border: 'border-line-soft', header: 'text-muted' },
} as const

function pct(v: number, total: number) {
  return `${(v / total) * 100}%`
}

/** TABLEタブの中身。monster_id で繋がるテーブルをER図として見せる。 */
export function SchemaDiagram() {
  return (
    <div className="relative h-full w-full">
      <p className="font-display mb-2 text-sm font-bold text-neon">&gt;_ ER DIAGRAM</p>
      <div className="relative" style={{ aspectRatio: `${CANVAS.w} / ${CANVAS.h}` }}>
        <svg
          viewBox={`0 0 ${CANVAS.w} ${CANVAS.h}`}
          preserveAspectRatio="none"
          className="absolute inset-0 h-full w-full"
        >
          {CONNECTORS.map((c, i) => (
            <line
              key={i}
              x1={c.x1}
              y1={c.y1}
              x2={c.x2}
              y2={c.y2}
              stroke="var(--color-neon)"
              strokeOpacity={0.6}
              strokeWidth={1.5}
            />
          ))}
        </svg>

        {ENTITIES.map((e) => {
          const accent = ACCENT[e.accent]
          return (
            <div
              key={e.name}
              className={`absolute rounded-md border bg-panel px-3 py-2.5 ${accent.border}`}
              style={{ left: pct(e.x, CANVAS.w), top: pct(e.y, CANVAS.h), width: pct(e.w, CANVAS.w) }}
            >
              <p className={`text-xs font-bold ${accent.header}`}>{e.name}</p>
              <div className="my-1 border-t border-line" />
              {e.columns.map(([col, type, key]) => (
                <div key={col} className="flex items-center justify-between gap-2 text-[10px]">
                  <span className={key ? 'text-neon' : 'text-muted'}>
                    {key ? `${col} (${key})` : col}
                  </span>
                  <span className="text-faint">{type}</span>
                </div>
              ))}
            </div>
          )
        })}
      </div>
    </div>
  )
}
