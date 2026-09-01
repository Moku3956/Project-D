import { useState } from 'react'
import { BattleMenu } from '../features/battle/BattleMenu'

/** 全画面共通のヘッダー。右端の三本線ボタンでバトルメニューを開く。 */
export function Header({ showMenu }: { showMenu: boolean }) {
  const [open, setOpen] = useState(false)

  return (
    <header className="relative flex items-center justify-between border-b border-line px-6 py-4">
      <div className="flex items-center gap-3">
        <span className="rounded border border-neon px-2 py-0.5 text-[11px] font-bold text-neon">
          SQL
        </span>
        <h1 className="font-display text-xl font-extrabold tracking-wide text-ink">SQL-MONSTER</h1>
      </div>

      {showMenu && (
        <>
          <button
            type="button"
            aria-label="メニュー"
            onClick={() => setOpen((v) => !v)}
            className="flex flex-col gap-[5px] p-2 hover:opacity-70"
          >
            <span className="block h-0.5 w-[18px] rounded bg-muted" />
            <span className="block h-0.5 w-[18px] rounded bg-muted" />
            <span className="block h-0.5 w-[18px] rounded bg-muted" />
          </button>
          {open && <BattleMenu onClose={() => setOpen(false)} />}
        </>
      )}
    </header>
  )
}
