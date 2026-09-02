import { useState } from 'react'
import { BattleMenu } from '../features/battle/BattleMenu'

/** 三本線ボタン。押すとBattleMenuを開く。バトル画面でフェーズタブの右隣に置く。 */
export function SettingsButton() {
  const [open, setOpen] = useState(false)

  return (
    <div className="relative shrink-0">
      <button
        type="button"
        aria-label="メニュー"
        onClick={() => setOpen((v) => !v)}
        className="flex flex-col gap-[5px] rounded-lg border border-line bg-panel p-3 hover:opacity-70"
      >
        <span className="block h-0.5 w-[18px] rounded bg-muted" />
        <span className="block h-0.5 w-[18px] rounded bg-muted" />
        <span className="block h-0.5 w-[18px] rounded bg-muted" />
      </button>
      {open && <BattleMenu onClose={() => setOpen(false)} />}
    </div>
  )
}
