import { useDbInternal } from '../../shared/store'
import { useI18n } from '../../shared/i18n'

/** セッション内の全テーブルを切り替えるタブ。CREATE TABLE文をエディターで
 * 実行すると、ここに新しいタブが増える(db-internal-app/docs/spec.md参照)。 */
export function TableTabs() {
  const tables = useDbInternal((s) => s.tables)
  const currentTable = useDbInternal((s) => s.currentTable)
  const switchTable = useDbInternal((s) => s.switchTable)
  const createTable = useDbInternal((s) => s.createTable)
  const busy = useDbInternal((s) => s.busy)
  const { t } = useI18n()

  if (tables.length === 0) return null

  return (
    <div className="flex flex-wrap gap-2 pb-4">
      {tables.map((table) => (
        <button
          key={table.name}
          type="button"
          disabled={busy}
          onClick={() => void switchTable(table.name)}
          className={`rounded-full px-4 py-2 text-xs font-bold transition disabled:opacity-50 ${
            table.name === currentTable
              ? 'bg-accent text-onaccent'
              : 'border border-line bg-surface text-muted hover:text-ink'
          }`}
        >
          {table.name}
        </button>
      ))}
      <button
        type="button"
        disabled={busy}
        onClick={() => void createTable()}
        title={t('newTableTitle')}
        className="rounded-full border border-dashed border-line px-4 py-2 text-xs font-bold text-muted transition hover:text-ink disabled:opacity-50"
      >
        {t('newTable')}
      </button>
    </div>
  )
}
