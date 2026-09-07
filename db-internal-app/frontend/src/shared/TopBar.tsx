import { useLocation, useNavigate } from 'react-router-dom'
import { useDbInternal } from './store'
import { useI18n } from './i18n'

export function TopBar() {
  const reset = useDbInternal((s) => s.reset)
  const busy = useDbInternal((s) => s.busy)
  const isHome = useLocation().pathname === '/'
  const navigate = useNavigate()
  const { t, locale, setLocale } = useI18n()

  return (
    <div className="flex items-center justify-between pb-6 pt-8">
      <div>
        {isHome ? (
          <h1 className="border-b-2 border-accent pb-1 font-sans text-xl font-bold text-ink">
            {t('appName')}
          </h1>
        ) : (
          <button
            type="button"
            onClick={() => navigate('/')}
            className="text-sm font-bold text-muted hover:text-ink"
          >
            {t('backToHome')}
          </button>
        )}
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
