import { useState } from 'react'
import { useDbInternal } from './store'
import { useI18n } from './i18n'

const DEFAULT_SEED_COUNT = 60

type SqlMode = 'INSERT' | 'UPDATE' | 'DELETE'
const MODES: SqlMode[] = ['INSERT', 'UPDATE', 'DELETE']

/** モード切り替えボタンで、選択中テーブルのカラムに沿ったテンプレートSQLを
 * 組み立てる。値は空クオートのままにしておき、ユーザーがその場で埋める。
 * PKは先頭カラムという、このアプリの他の便宜機能(paddedSeedId等)と同じ
 * 前提を使う。 */
function buildTemplate(mode: SqlMode, table: string, columns: string[]): string {
  const cols = columns.length > 0 ? columns : ['id', 'name']
  const pk = cols[0]
  if (mode === 'INSERT') {
    return `INSERT INTO ${table} VALUES (${cols.map(() => "''").join(', ')})`
  }
  if (mode === 'DELETE') {
    return `DELETE FROM ${table} WHERE ${pk} = ''`
  }
  const rest = cols.slice(1)
  const sets = (rest.length > 0 ? rest : ['name']).map((c) => `${c} = ''`).join(', ')
  return `UPDATE ${table} SET ${sets} WHERE ${pk} = ''`
}

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
  const currentTable = useDbInternal((s) => s.currentTable)
  const tables = useDbInternal((s) => s.tables)
  const [seedCount, setSeedCount] = useState(DEFAULT_SEED_COUNT)
  const [mode, setMode] = useState<SqlMode>('INSERT')
  const { t } = useI18n()

  const columns = tables.find((tbl) => tbl.name === currentTable)?.columns ?? []

  const applyMode = (m: SqlMode) => {
    setMode(m)
    setSql(buildTemplate(m, currentTable, columns))
  }

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
            {MODES.map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => applyMode(m)}
                className={`rounded-full px-2.5 py-1.5 ${mode === m ? 'bg-accent text-onaccent' : 'text-muted hover:text-ink'}`}
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
            placeholder={t('editorPlaceholder')}
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
