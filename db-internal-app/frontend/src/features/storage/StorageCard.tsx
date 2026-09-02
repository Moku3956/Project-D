import { useState } from 'react'
import { useDbInternal } from '../../shared/store'
import { TreeDiagram } from './TreeDiagram'
import { ZoomPanCanvas } from './ZoomPanCanvas'
import { IDENTITY_TRANSFORM, type ZoomTransform } from './zoom'
import { EXAMPLE_TREE } from './exampleTree'

const EMPTY_NEW_PKS = new Set<unknown>()

type Mode = 'real' | 'example'

export function StorageCard() {
  const liveTree = useDbInternal((s) => s.tree)
  const newPKs = useDbInternal((s) => s.newPKs)
  const [mode, setMode] = useState<Mode>('real')
  const [expanded, setExpanded] = useState(false)
  const [transform, setTransform] = useState<ZoomTransform>(IDENTITY_TRANSFORM)

  const tree = mode === 'real' ? liveTree : EXAMPLE_TREE
  const highlightedPKs = mode === 'real' ? newPKs : EMPTY_NEW_PKS

  return (
    <div className="rounded-2xl border border-line bg-surface p-7 shadow-sm">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-bold text-ink">B+Tree ページ構造</h2>
          <p className="pt-1 text-xs text-muted">本来は、すごい平べったいです！</p>
        </div>
        {tree && (
          <button
            type="button"
            onClick={() => {
              setTransform(IDENTITY_TRANSFORM)
              setExpanded(true)
            }}
            className="flex items-center gap-1.5 rounded-full bg-accent px-3 py-2 text-xs font-bold text-onaccent hover:brightness-110"
          >
            ⤢ 拡大表示
          </button>
        )}
      </div>

      <div className="mt-3 flex gap-1.5 rounded-full bg-bg p-1 text-xs font-bold">
        <button
          type="button"
          onClick={() => setMode('real')}
          className={`flex-1 rounded-full py-1.5 ${mode === 'real' ? 'bg-accent text-onaccent' : 'text-muted hover:text-ink'}`}
        >
          実データ
        </button>
        <button
          type="button"
          onClick={() => setMode('example')}
          className={`flex-1 rounded-full py-1.5 ${mode === 'example' ? 'bg-accent text-onaccent' : 'text-muted hover:text-ink'}`}
        >
          イメージ図(例)
        </button>
      </div>
      {mode === 'example' && (
        <p className="pt-2 text-[11px] text-muted">
          ※ 今のDBの中身とは無関係の作り物です。データが増えて何段にも育つとこんな形になる、という例です。
        </p>
      )}

      <div className="overflow-x-auto pt-3">
        {tree ? (
          <TreeDiagram tree={tree} newPKs={highlightedPKs} />
        ) : (
          <p className="pt-4 text-sm text-muted">読み込み中…</p>
        )}
      </div>

      {expanded && tree && (
        <div className="fixed inset-0 z-50 flex flex-col bg-surface">
          <div className="flex shrink-0 items-center justify-between border-b border-line px-8 py-5">
            <h2 className="text-lg font-bold text-ink">
              B+Tree ページ構造(拡大表示){mode === 'example' && ' — イメージ図(例)'}
            </h2>
            <div className="flex items-center gap-2">
              <span className="pr-2 text-xs text-muted">ホイールで拡大縮小・ドラッグで移動できます</span>
              <button
                type="button"
                onClick={() => setTransform(IDENTITY_TRANSFORM)}
                className="rounded-full border border-line px-3.5 py-2 text-xs font-bold text-muted hover:text-ink"
              >
                ↺ もとに戻す
              </button>
              <button
                type="button"
                onClick={() => setExpanded(false)}
                className="rounded-full bg-accent px-3.5 py-2 text-xs font-bold text-onaccent hover:brightness-110"
              >
                閉じる ✕
              </button>
            </div>
          </div>
          <div className="min-h-0 flex-1">
            <ZoomPanCanvas transform={transform} onTransformChange={setTransform}>
              <TreeDiagram tree={tree} newPKs={highlightedPKs} />
            </ZoomPanCanvas>
          </div>
        </div>
      )}
    </div>
  )
}
