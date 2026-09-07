import { useI18n } from '../../shared/i18n'

/** WAL可視化ページのプレースホルダー。詳細な表示内容(ログ一覧・クラッシュ/
 * リカバリー再現など)はユーザーとの追加の仕様確認後に実装する。 */
export function WalCard() {
  const { t } = useI18n()

  return (
    <div className="rounded-2xl border border-line bg-surface p-7 shadow-sm">
      <h2 className="text-base font-bold text-ink">{t('walPlaceholderTitle')}</h2>
      <p className="pt-1 text-xs text-muted">{t('walPlaceholderDesc')}</p>
    </div>
  )
}
