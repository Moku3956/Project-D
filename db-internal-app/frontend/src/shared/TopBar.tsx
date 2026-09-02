import { useDbInternal } from './store'

export function TopBar() {
  const reset = useDbInternal((s) => s.reset)
  const busy = useDbInternal((s) => s.busy)

  return (
    <div className="flex items-center justify-between pb-6 pt-8">
      <div>
        <div className="flex items-center gap-2">
          <span className="h-2.5 w-2.5 rounded-full bg-accent" />
          <h1 className="font-sans text-xl font-bold text-ink">db-internal-app</h1>
        </div>
        <p className="pt-1 text-xs text-muted">SQLの内部を、1ステージずつのぞく</p>
      </div>
      <button
        type="button"
        disabled={busy}
        onClick={() => void reset()}
        className="rounded-full border border-line px-4 py-2 text-xs font-bold text-muted hover:text-ink disabled:opacity-50"
      >
        リセット
      </button>
    </div>
  )
}
