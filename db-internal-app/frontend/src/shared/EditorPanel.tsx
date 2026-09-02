import { useState } from 'react'
import { useDbInternal } from './store'

const DEFAULT_SEED_COUNT = 60

export function EditorPanel() {
  const sql = useDbInternal((s) => s.sql)
  const setSql = useDbInternal((s) => s.setSql)
  const run = useDbInternal((s) => s.run)
  const seedMany = useDbInternal((s) => s.seedMany)
  const busy = useDbInternal((s) => s.busy)
  const error = useDbInternal((s) => s.error)
  const [seedCount, setSeedCount] = useState(DEFAULT_SEED_COUNT)

  return (
    <div className="w-[432px] shrink-0 rounded-2xl border border-line bg-surface p-6 shadow-sm">
      <h2 className="pb-3 text-base font-bold text-ink">エディター</h2>

      <textarea
        value={sql}
        onChange={(e) => setSql(e.target.value)}
        spellCheck={false}
        className="sql-input h-56 w-full resize-none rounded-xl border border-line bg-bg p-4 text-sm text-ink outline-none focus:border-accent"
      />

      <button
        type="button"
        disabled={busy}
        onClick={() => void run()}
        className="mt-3.5 w-full rounded-xl bg-accent py-3.5 text-sm font-bold text-onaccent hover:brightness-110 disabled:opacity-50"
      >
        {busy ? '実行中…' : '実行 ▸'}
      </button>

      <div className="mt-5 rounded-xl border border-dashed border-accent2/50 bg-orange-50/60 p-3.5">
        <p className="pb-2 text-[11px] font-bold text-accent2">⚡ テスト用: まとめてダミー行を投入</p>
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
            title="idを自動採番してダミー行をまとめてINSERTする(1ページに収まらない件数を手でクリックせず試すための機能。エディターのSQLは実行しない)"
          >
            件まとめてINSERT ⚡
          </button>
        </div>
        <p className="pt-2 text-[10px] leading-relaxed text-muted">
          わざと長いidを入れて1ページ数件しか入らないようにしています(表示は簡潔にしたまま)。idは複合キーとして葉・内部ノード両方に含まれるため、Rootだけが極端に枝分かれすることもありません。60件で葉2件・内部ノード4〜7件ずつの多段構造になります(実測値)。
        </p>
      </div>

      {error && (
        <p className="mt-3 rounded-lg bg-orange-50 p-3 text-xs text-accent2" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}
