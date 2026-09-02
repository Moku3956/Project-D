import { useState } from 'react'
import { useDbInternal } from '../../shared/store'
import { TreeDiagram } from './TreeDiagram'

export function StorageCard() {
  const tree = useDbInternal((s) => s.tree)
  const newPKs = useDbInternal((s) => s.newPKs)
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="rounded-2xl border border-line bg-surface p-7 shadow-sm">
      <div className="flex items-center justify-between">
        <h2 className="text-base font-bold text-ink">B+Tree ページ構造</h2>
        {tree && (
          <button
            type="button"
            onClick={() => setExpanded(true)}
            className="flex items-center gap-1.5 rounded-full bg-accent px-3 py-2 text-xs font-bold text-onaccent hover:brightness-110"
          >
            ⤢ 拡大表示
          </button>
        )}
      </div>

      <div className="pt-2">
        {tree ? (
          <TreeDiagram tree={tree} newPKs={newPKs} />
        ) : (
          <p className="pt-4 text-sm text-muted">
            まだテーブルがありません。右のエディターでCREATE TABLEを実行してください。
          </p>
        )}
      </div>

      {expanded && tree && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-8"
          onClick={() => setExpanded(false)}
        >
          <div
            className="max-h-full max-w-full overflow-auto rounded-2xl bg-surface p-8 shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between pb-4">
              <h2 className="text-lg font-bold text-ink">B+Tree ページ構造(拡大表示)</h2>
              <button
                type="button"
                onClick={() => setExpanded(false)}
                className="rounded-full border border-line px-3 py-1.5 text-xs font-bold text-muted hover:text-ink"
              >
                閉じる ✕
              </button>
            </div>
            <TreeDiagram tree={tree} newPKs={newPKs} scale={1.6} />
          </div>
        </div>
      )}
    </div>
  )
}
