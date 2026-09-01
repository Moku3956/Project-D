/** 画面下部の対応表。SQLの操作と技の種類の対応(spec.md「②攻撃」)。 */
const OPS = [
  { op: 'SELECT', color: 'var(--color-neon)', desc: '= Intelligence & Enemy Patterns' },
  { op: 'UPDATE', color: 'var(--color-amber)', desc: '= Direct Physical Attack' },
  { op: 'INSERT', color: 'var(--color-done)', desc: '= Inject Status Effect / Debuff' },
  { op: 'DELETE', color: 'var(--color-danger)', desc: '= Remove Monster Buffs' },
]

export function SyntaxMap() {
  return (
    <footer className="mx-6 flex flex-wrap items-center gap-x-8 gap-y-2 rounded-xl border border-line bg-panel px-4 py-3">
      <span className="text-[13px] font-bold text-neon">SQL OPERATION SYNTAX MAP:</span>
      {OPS.map(({ op, color, desc }) => (
        <span key={op} className="flex items-center gap-2">
          <span
            className="rounded px-2 py-0.5 text-[11px] font-bold text-base"
            style={{ background: color }}
          >
            {op}
          </span>
          <span className="text-[13px] text-muted">{desc}</span>
        </span>
      ))}
    </footer>
  )
}
