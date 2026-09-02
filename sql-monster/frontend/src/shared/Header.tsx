/** ホーム画面用のロゴ表示。バトル画面はheader-barを使わない構成になったため、ここでは使わない。 */
export function Header() {
  return (
    <header className="flex items-center gap-3 border-b border-line px-6 py-4">
      <span className="rounded border border-neon px-2 py-0.5 text-[11px] font-bold text-neon">
        SQL
      </span>
      <h1 className="font-display text-xl font-extrabold tracking-wide text-ink">SQL-MONSTER</h1>
    </header>
  )
}
