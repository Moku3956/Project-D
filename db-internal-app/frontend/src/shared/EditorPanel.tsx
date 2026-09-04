import { useState } from 'react'
import { useDbInternal } from './store'
import { useI18n } from './i18n'

const DEFAULT_SEED_COUNT = 60

export function EditorPanel() {
  const sql = useDbInternal((s) => s.sql)
  const setSql = useDbInternal((s) => s.setSql)
  const run = useDbInternal((s) => s.run)
  const seedMany = useDbInternal((s) => s.seedMany)
  const busy = useDbInternal((s) => s.busy)
  const error = useDbInternal((s) => s.error)
  const [seedCount, setSeedCount] = useState(DEFAULT_SEED_COUNT)
  const { t } = useI18n()

  return (
    <div className="w-[432px] shrink-0 rounded-2xl border border-line bg-surface p-6 shadow-sm">
      <h2 className="pb-3 text-base font-bold text-ink">{t('editorTitle')}</h2>

      <textarea
        value={sql}
        onChange={(e) => setSql(e.target.value)}
        spellCheck={false}
        placeholder={t('editorPlaceholder')}
        className="sql-input h-56 w-full resize-none rounded-xl border border-line bg-bg p-4 text-sm text-ink outline-none focus:border-accent"
      />

      <button
        type="button"
        disabled={busy}
        onClick={() => void run()}
        className="mt-3.5 w-full rounded-xl bg-accent py-3.5 text-sm font-bold text-onaccent hover:brightness-110 disabled:opacity-50"
      >
        {busy ? t('running') : t('run')}
      </button>

      <div className="mt-5 rounded-xl border border-dashed border-accent2/50 bg-orange-50/60 p-3.5">
        <p className="pb-2 text-[11px] font-bold text-accent2">{t('seedTitle')}</p>
        <div className="flex items-center gap-2">
          <input
            type="number"
            min={1}
            max={2000}
            value={seedCount}
            onChange={(e) => setSeedCount(Number(e.target.value))}
            className="w-20 rounded-lg border border-accent2/40 bg-surface px-2 py-2 text-xs text-ink outline-none focus:border-accent2"
          />
          <button
            type="button"
            disabled={busy}
            onClick={() => void seedMany(seedCount)}
            className="flex-1 rounded-lg bg-accent2 py-2 text-xs font-bold text-onaccent hover:brightness-110 disabled:opacity-50"
            title={t('seedButtonTitle')}
          >
            {t('seedButton')}
          </button>
        </div>
        <p className="pt-2 text-[10px] leading-relaxed text-muted">{t('seedHint')}</p>
      </div>

      {error && (
        <p className="mt-3 rounded-lg bg-orange-50 p-3 text-xs text-accent2" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}
