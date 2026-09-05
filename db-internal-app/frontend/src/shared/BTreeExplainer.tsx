import { useState } from 'react'
import { useI18n } from './i18n'

/** B+Tree自体の簡単な説明。常時表示のカードだと画面を占有しすぎる
 * (ユーザー指摘)ため、小さいボタン+タップで開くモーダルにした。 */
export function BTreeExplainer() {
  const [open, setOpen] = useState(false)
  const { t } = useI18n()

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="mb-6 rounded-full border border-line bg-surface px-4 py-2 text-xs font-bold text-muted hover:text-ink"
      >
        {t('btreeExplainerTitle')}
      </button>

      {open && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-6"
          onClick={() => setOpen(false)}
        >
          <div
            className="max-w-md rounded-2xl border border-line bg-surface p-6 shadow-lg"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between gap-4">
              <h2 className="text-sm font-bold text-ink">{t('btreeExplainerTitle')}</h2>
              <button
                type="button"
                onClick={() => setOpen(false)}
                className="shrink-0 rounded-full bg-accent px-3 py-1.5 text-xs font-bold text-onaccent hover:brightness-110"
              >
                {t('close')}
              </button>
            </div>
            <p className="pt-3 text-xs leading-relaxed text-muted">{t('btreeExplainerBody')}</p>
          </div>
        </div>
      )}
    </>
  )
}
