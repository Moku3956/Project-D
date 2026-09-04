import { useDbInternal } from './store'
import { useI18n } from './i18n'

export function TopBar() {
  const reset = useDbInternal((s) => s.reset)
  const busy = useDbInternal((s) => s.busy)
  const { t, locale, setLocale } = useI18n()

  return (
    <div className="flex items-center justify-between pb-6 pt-8">
      <div>
        <div className="flex items-center gap-2">
          <span className="h-2.5 w-2.5 rounded-full bg-accent" />
          <h1 className="font-sans text-xl font-bold text-ink">db-internal-app</h1>
        </div>
        <p className="pt-1 text-xs text-muted">{t('appSubtitle')}</p>
      </div>
      <div className="flex items-center gap-3">
        <div className="flex rounded-full bg-bg p-1 text-xs font-bold">
          <button
            type="button"
            onClick={() => setLocale('ja')}
            aria-pressed={locale === 'ja'}
            className={`rounded-full px-3 py-1.5 ${locale === 'ja' ? 'bg-accent text-onaccent' : 'text-muted hover:text-ink'}`}
          >
            JA
          </button>
          <button
            type="button"
            onClick={() => setLocale('en')}
            aria-pressed={locale === 'en'}
            className={`rounded-full px-3 py-1.5 ${locale === 'en' ? 'bg-accent text-onaccent' : 'text-muted hover:text-ink'}`}
          >
            EN
          </button>
        </div>
        <button
          type="button"
          disabled={busy}
          onClick={() => void reset()}
          className="rounded-full border border-line px-4 py-2 text-xs font-bold text-muted hover:text-ink disabled:opacity-50"
        >
          {t('reset')}
        </button>
      </div>
    </div>
  )
}
