import { useDbInternal } from '../../shared/store'

/** セッション内の全テーブルを切り替えるタブ。CREATE TABLE文をエディターで
 * 実行すると、ここに新しいタブが増える(db-internal-app/docs/spec.md参照)。 */
const CREATE_TABLE_TEMPLATE =
  "CREATE TABLE new_table (id VARCHAR(600) PRIMARY KEY, name VARCHAR(50))"

export function TableTabs() {
  const tables = useDbInternal((s) => s.tables)
  const currentTable = useDbInternal((s) => s.currentTable)
  const switchTable = useDbInternal((s) => s.switchTable)
  const setSql = useDbInternal((s) => s.setSql)
  const busy = useDbInternal((s) => s.busy)

  if (tables.length === 0) return null

  return (
    <div className="flex flex-wrap gap-2 pb-4">
      {tables.map((t) => (
        <button
          key={t.name}
          type="button"
          disabled={busy}
          onClick={() => void switchTable(t.name)}
          className={`rounded-full px-4 py-2 text-xs font-bold transition disabled:opacity-50 ${
            t.name === currentTable
              ? 'bg-accent text-onaccent'
              : 'border border-line bg-surface text-muted hover:text-ink'
          }`}
        >
          {t.name}
        </button>
      ))}
      <button
        type="button"
        disabled={busy}
        onClick={() => setSql(CREATE_TABLE_TEMPLATE)}
        title="エディターにCREATE TABLE文のひな形を入力します。実行すると新しいタブが増えます。"
        className="rounded-full border border-dashed border-line px-4 py-2 text-xs font-bold text-muted transition hover:text-ink disabled:opacity-50"
      >
        + 新しいテーブル
      </button>
    </div>
  )
}
