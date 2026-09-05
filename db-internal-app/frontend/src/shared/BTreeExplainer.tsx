import { useI18n } from './i18n'

/** ページ上部に置く、B+Tree自体の簡単な説明欄(ユーザー指示)。既存のカードと
 * 同じ見た目(角丸・枠線・白背景)に揃えている。 */
export function BTreeExplainer() {
  const { t } = useI18n()

  return (
    <div className="mb-6 rounded-2xl border border-line bg-surface p-6 shadow-sm">
      <h2 className="text-sm font-bold text-ink">{t('btreeExplainerTitle')}</h2>
      <p className="pt-2 text-xs leading-relaxed text-muted">{t('btreeExplainerBody')}</p>
    </div>
  )
}
