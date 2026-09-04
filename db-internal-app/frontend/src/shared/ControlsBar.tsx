import { useEffect, useState } from 'react'
import { useDbInternal } from './store'
import { useI18n } from './i18n'
import { SQL_MODES } from './sqlTemplates'

const DEFAULT_SEED_COUNT = 20

/** btree.appを参考に、上部に横並びの1つのツールバー(ランダム追加・まとめて
 * 追加・エディター)としてまとめた(ユーザー指示)。エディターは複数行の
 * テキストエリアではなく1行入力にし、INSERT/UPDATE/DELETEの切り替えボタンで
 * その場でテンプレートを差し込む形にした(「エディターはあまり使わなさそう
 * だから、1行書けるスペースがあればいい」というユーザー指示による)。 */
export function ControlsBar() {
  const sql = useDbInternal((s) => s.sql)
  const setSql = useDbInternal((s) => s.setSql)
  const run = useDbInternal((s) => s.run)
  const seedMany = useDbInternal((s) => s.seedMany)
  const busy = useDbInternal((s) => s.busy)
  const error = useDbInternal((s) => s.error)
  const tables = useDbInternal((s) => s.tables)
  const sqlMode = useDbInternal((s) => s.sqlMode)
  const applySqlMode = useDbInternal((s) => s.applySqlMode)
  const [seedCount, setSeedCount] = useState(DEFAULT_SEED_COUNT)
  const { t } = useI18n()

  // 空欄のプレースホルダーではなく、最初から実際に実行できるクエリを入れて
  // おく(「最初からクエリぶち込んで」というユーザー指示による)。tablesの
  // 読み込み完了(columnsが分かる)を待って一度だけ埋める。
  useEffect(() => {
    if (sql === '' && tables.length > 0) {
      applySqlMode('INSERT')
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tables.length])

  return (
    <div className="mb-6 flex flex-wrap items-start gap-6 rounded-2xl border border-line bg-surface p-6 shadow-sm">
      <div className="flex flex-col gap-2">
        <h2 className="text-xs font-bold text-muted">{t('addRandom')}</h2>
        <button
          type="button"
          disabled={busy}
          onClick={() => void seedMany(1)}
          title={t('addRandomTitle')}
          className="rounded-lg bg-accent2 px-4 py-2.5 text-xs font-bold text-onaccent hover:brightness-110 disabled:opacity-50"
        >
          🎲 {t('addRandom')}
        </button>
      </div>

      <div className="h-14 w-px shrink-0 bg-line" />

      <div className="flex flex-col gap-2">
        <h2 className="text-xs font-bold text-muted">{t('bulkInsert')}</h2>
        <div className="flex items-center gap-2">
          <input
            type="number"
            min={1}
            max={2000}
            value={seedCount}
            onChange={(e) => setSeedCount(Number(e.target.value))}
            className="w-20 rounded-lg border border-line bg-bg px-2 py-2 text-xs text-ink outline-none focus:border-accent"
          />
          <button
            type="button"
            disabled={busy}
            onClick={() => void seedMany(seedCount)}
            title={t('bulkInsertTitle')}
            className="rounded-lg bg-accent2 px-3 py-2 text-xs font-bold text-onaccent hover:brightness-110 disabled:opacity-50"
          >
            {t('bulkInsert')} ⚡
          </button>
        </div>
      </div>

      <div className="h-14 w-px shrink-0 bg-line" />

      <div className="flex min-w-[380px] flex-1 flex-col gap-2">
        <h2 className="text-xs font-bold text-muted">{t('editorTitle')}</h2>
        <div className="flex items-center gap-2">
          <div className="flex shrink-0 gap-1 rounded-full bg-bg p-1 text-[11px] font-bold">
            {SQL_MODES.map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => applySqlMode(m)}
                className={`rounded-full px-2.5 py-1.5 ${sqlMode === m ? 'bg-accent text-onaccent' : 'text-muted hover:text-ink'}`}
              >
                {m}
              </button>
            ))}
          </div>
          <input
            type="text"
            value={sql}
            onChange={(e) => setSql(e.target.value)}
            spellCheck={false}
            className="sql-input min-w-0 flex-1 rounded-lg border border-line bg-bg px-3 py-2 text-xs text-ink outline-none focus:border-accent"
          />
          <button
            type="button"
            disabled={busy}
            onClick={() => void run()}
            className="shrink-0 rounded-lg bg-accent px-4 py-2 text-xs font-bold text-onaccent hover:brightness-110 disabled:opacity-50"
          >
            {busy ? t('running') : t('run')}
          </button>
        </div>
      </div>

      {error && (
        <p className="w-full rounded-lg bg-orange-50 p-3 text-xs text-accent2" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}
