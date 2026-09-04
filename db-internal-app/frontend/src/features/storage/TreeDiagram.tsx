import { CELL_W, LEAF_LABEL_H, layoutTree, type Layout } from './layout'
import type { TreeSnapshot } from '../../shared/types'

const PADDING = 16

function LeafBox({ node, tableName }: { node: Layout['nodes'][number]; tableName: string }) {
  // ページはこのテーブル専用ではなく、物理B+Tree全体を全テーブルで共有している
  // (docs/spec.md「Storage」参照)。DumpTreeは選択中テーブルの行だけをRowsに
  // 残すため、cellsが空のページは「このテーブルのデータがない」だけで、実際は
  // 他テーブルの行で埋まっている可能性が高い。そのためラベルは実際に選択中
  // テーブルの行を含むページにだけ付け、空のページには付けない
  // (「リーフノードのすべてに新しいテーブルの名前が書かれる」というユーザー
  // 指摘の修正)。
  const hasOwnRows = (node.cells ?? []).length > 0
  return (
    <div
      className="absolute flex flex-col overflow-hidden rounded-xl border border-line bg-surface shadow-sm"
      style={{
        left: node.x - node.width / 2 + PADDING,
        top: node.y + PADDING,
        width: node.width,
        height: node.height,
      }}
    >
      <div
        className="flex shrink-0 items-center justify-center truncate border-b border-line bg-bg px-1 text-[9px] font-bold text-muted"
        style={{ height: LEAF_LABEL_H }}
        title={hasOwnRows ? tableName : '他テーブルの行を含む可能性があるページ'}
      >
        {hasOwnRows ? tableName : '?'}
      </div>
      <div className="flex flex-1 overflow-hidden">
        {(node.cells ?? []).map((cell, i) =>
          cell.kind === 'omit' ? (
            <div
              key={`omit-${i}`}
              className="flex shrink-0 items-center justify-center bg-accent text-[10px] font-bold text-onaccent"
              style={{ width: 56 }}
            >
              …{cell.count}件…
            </div>
          ) : (
            <div
              key={i}
              className={`flex shrink-0 items-center justify-center border-l border-line px-1 font-mono text-[11px] first:border-l-0 ${
                cell.isNew ? 'bg-orange-50 font-bold text-accent2' : 'bg-bg text-ink'
              }`}
              style={{ width: CELL_W }}
              title={cell.text}
            >
              <span className="truncate">{cell.text}</span>
            </div>
          ),
        )}
        {(node.cells ?? []).length === 0 && (
          <div className="flex w-full items-center justify-center text-[11px] text-muted">(空)</div>
        )}
      </div>
    </div>
  )
}

function InternalBox({ node, isRoot }: { node: Layout['nodes'][number]; isRoot: boolean }) {
  return (
    <div
      className="absolute flex items-center justify-center rounded-xl border-[1.5px] border-accent bg-bg px-3 text-sm font-bold text-ink"
      style={{ left: node.x - node.width / 2 + PADDING, top: node.y + PADDING, width: node.width, height: node.height }}
    >
      {isRoot ? 'Root' : `Page ${node.pageId}`}
    </div>
  )
}

/** B+Treeのツリー図。db-internal-app/docs/spec.md「Storage」の確定デザイン
 * (直線接続・分岐条件は横線上・葉はKVを横並びセルで表現)を、任意の階層数・
 * 子ノード数に対応する形で描画する。等倍の生の大きさで描画するだけで、拡大縮小・
 * スクロールは呼び出し側(StorageCard)が担当する。 */
export function TreeDiagram({
  tree,
  newPKs,
  tableName,
}: {
  tree: TreeSnapshot
  newPKs: Set<unknown>
  tableName: string
}) {
  const layout = layoutTree(tree, newPKs)
  const width = layout.width + PADDING * 2
  const height = layout.height + PADDING * 2

  return (
    <div className="relative" style={{ width, height }}>
      <svg
        className="pointer-events-none absolute inset-0"
        width={width}
        height={height}
        viewBox={`0 0 ${width} ${height}`}
      >
        {layout.edges.map((e, i) => (
          <g key={i}>
            <line
              x1={e.fromX + PADDING}
              y1={e.fromY + PADDING}
              x2={e.toX + PADDING}
              y2={e.toY + PADDING}
              stroke="#2FB4FF"
              strokeWidth={2}
            />
            <circle cx={e.fromX + PADDING} cy={e.fromY + PADDING} r={3} fill="#2FB4FF" />
            <circle cx={e.toX + PADDING} cy={e.toY + PADDING} r={3} fill="#2FB4FF" />
          </g>
        ))}
      </svg>

      {layout.edges.map(
        (e, i) =>
          e.label && (
            <div
              key={i}
              className="absolute -translate-x-1/2 -translate-y-1/2 rounded-full border border-accent bg-surface px-2 py-0.5 text-[10px] font-bold text-accent"
              style={{ left: (e.fromX + e.toX) / 2 + PADDING, top: (e.fromY + e.toY) / 2 + PADDING }}
            >
              {e.label}
            </div>
          ),
      )}

      {layout.nodes.map((n) =>
        n.isLeaf ? (
          <LeafBox key={n.pageId} node={n} tableName={tableName} />
        ) : (
          <InternalBox key={n.pageId} node={n} isRoot={n.pageId === tree.rootPageId} />
        ),
      )}
    </div>
  )
}
