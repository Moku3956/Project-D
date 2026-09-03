import { useState } from 'react'
import { useDbInternal } from '../../shared/store'
import { TreeDiagram } from './TreeDiagram'
import { FitToScreen } from './FitToScreen'
import { DataTable } from './DataTable'

export function StorageCard() {
  const tree = useDbInternal((s) => s.tree)
  const newPKs = useDbInternal((s) => s.newPKs)
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="flex flex-col gap-5">
      <div className="rounded-2xl border border-line bg-surface p-7 shadow-sm">
        <div>
          <h2 className="text-base font-bold text-ink">テーブル(users)</h2>
          <p className="pt-1 text-xs text-muted">見た目はただのテーブルですが…</p>
        </div>
        <div className="pt-3">
          {tree ? <DataTable tree={tree} /> : <p className="pt-1 text-sm text-muted">読み込み中…</p>}
        </div>
      </div>

      <div className="rounded-2xl border border-line bg-surface p-7 shadow-sm">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-base font-bold text-ink">B+Tree ページ構造</h2>
            <p className="pt-1 text-xs text-muted">実際はこうやって保存されています。本来は、すごい平べったいです！</p>
          </div>
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

        <div className="overflow-x-auto pt-3">
          {tree ? (
            <TreeDiagram tree={tree} newPKs={newPKs} />
          ) : (
            <p className="pt-4 text-sm text-muted">読み込み中…</p>
          )}
        </div>

        {expanded && tree && (
          <div className="fixed inset-0 z-50 flex flex-col bg-surface">
            <div className="flex shrink-0 items-center justify-between border-b border-line px-8 py-5">
              <h2 className="text-lg font-bold text-ink">B+Tree ページ構造(拡大表示)</h2>
              <button
                type="button"
                onClick={() => setExpanded(false)}
                className="rounded-full bg-accent px-3.5 py-2 text-xs font-bold text-onaccent hover:brightness-110"
              >
                閉じる ✕
              </button>
            </div>
            <div className="min-h-0 flex-1">
              <FitToScreen>
                <TreeDiagram tree={tree} newPKs={newPKs} />
              </FitToScreen>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
