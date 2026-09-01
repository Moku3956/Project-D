import { useGame } from '../../shared/store'

/**
 * ヘッダーの三本線から開くメニュー。中身は音量や表示設定ではなく、
 * バトルの進行そのものを操作する項目だけ(docs/battle_screen_ui.md)。
 */
export function BattleMenu({ onClose }: { onClose: () => void }) {
  const restart = useGame((s) => s.restart)
  const quit = useGame((s) => s.quit)
  const backToHome = useGame((s) => s.backToHome)

  const items = [
    { label: 'RESUME', color: 'text-neon', action: onClose },
    {
      label: 'RESTART',
      color: 'text-muted',
      action: async () => {
        await restart()
        onClose()
      },
    },
    {
      label: 'QUIT',
      color: 'text-danger',
      action: async () => {
        // QUITはその対戦を敗北扱いにしてからホームへ戻る
        await quit()
        await backToHome()
        onClose()
      },
    },
  ]

  return (
    <div className="absolute right-6 top-full z-20 mt-2 w-[340px] rounded-2xl border border-line bg-panel">
      <div className="flex items-center justify-between border-b border-line px-5 py-4">
        <span className="font-display text-sm font-bold text-neon">&gt;_ MENU</span>
        <button type="button" onClick={onClose} className="text-muted hover:text-ink">
          x
        </button>
      </div>
      {items.map((item, i) => (
        <button
          key={item.label}
          type="button"
          onClick={item.action}
          className={`flex w-full items-center gap-3 px-5 py-4 text-left hover:bg-white/5 ${
            i > 0 ? 'border-t border-line' : ''
          }`}
        >
          <span className={`font-display text-sm font-bold ${item.color}`}>&gt;</span>
          <span className={`font-display text-sm font-bold ${item.color}`}>{item.label}</span>
        </button>
      ))}
    </div>
  )
}
