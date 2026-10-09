import { useNavigate } from 'react-router-dom'
import { useI18n } from '../../shared/i18n'

/** 起動時に最初に表示するホーム画面。ここから各可視化画面(B+Tree/WAL)
 * へ進む。左上のロゴ(TopBar)からもいつでもここに戻れる。 */
export function HomeCard() {
  const navigate = useNavigate()
  const { t } = useI18n()

  return (
    <div className="grid gap-4 pb-12 pt-8 sm:grid-cols-2">
      <button
        type="button"
        onClick={() => navigate('/storage')}
        className="rounded-2xl border border-line bg-surface p-7 text-left shadow-sm hover:border-accent"
      >
        <h2 className="text-base font-bold text-ink">{t('homeStorageTitle')}</h2>
        <p className="pt-1 text-xs text-muted">{t('homeStorageDesc')}</p>
      </button>
      <button
        type="button"
        onClick={() => navigate('/wal')}
        className="rounded-2xl border border-line bg-surface p-7 text-left shadow-sm hover:border-accent"
      >
        <h2 className="text-base font-bold text-ink">{t('homeWalTitle')}</h2>
        <p className="pt-1 text-xs text-muted">{t('homeWalDesc')}</p>
      </button>
    </div>
  )
}
