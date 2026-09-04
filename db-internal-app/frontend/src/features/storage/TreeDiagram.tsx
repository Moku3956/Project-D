import { CELL_W, LEAF_LABEL_H, layoutTree, type Layout } from './layout'
import type { TreeSnapshot } from '../../shared/types'
import { useI18n } from '../../shared/i18n'

const PADDING = 16

/** 1つの物理B+Treeを全テーブルで共有しているため、1ページの中に複数テーブルの
 * 行が混在することがある(ユーザー指摘「同じページの中にも違うテーブルの
 * レコードが含まれることがある」)。node.groupsは同テーブルの行が連続する
 * 区間ごとに分けられており、グループごとに独立したラベル+セル列を描画する。
 * 選択中テーブル以外のグループは淡色にして「他テーブルの行」だと分かるように
 * する。 */
function LeafBox({ node, currentTable }: { node: Layout['nodes'][number]; currentTable: string }) {
  const groups = node.groups ?? []
  const { t } = useI18n()
  return (
    <div
      className="absolute flex overflow-hidden rounded-xl border border-line bg-surface shadow-sm"
      style={{
        left: node.x - node.width / 2 + PADDING,
        top: node.y + PADDING,
        width: node.width,
        height: node.height,
      }}
    >
      {groups.length === 0 ? (
        <div className="flex w-full flex-col">
          <div className="shrink-0" style={{ height: LEAF_LABEL_H }} />
          <div className="flex flex-1 items-center justify-center text-[11px] text-muted">{t('treeEmpty')}</div>
        </div>
      ) : (
        groups.map((g, gi) => {
          const isCurrent = g.table === currentTable
          return (
            <div
              key={gi}
              className={`flex shrink-0 flex-col ${gi > 0 ? 'border-l-2 border-ink/15' : ''}`}
            >
              <div
                className={`flex shrink-0 items-center justify-center truncate border-b px-1 text-[9px] font-bold ${
                  isCurrent ? 'border-line bg-bg text-muted' : 'border-line bg-bg/60 text-muted/70'
                }`}
                style={{ height: LEAF_LABEL_H }}
                title={isCurrent ? g.table : t('treeOtherTableTitle', { table: g.table })}
              >
                {g.table}
              </div>
              <div className="flex flex-1 overflow-hidden">
                {g.cells.map((cell, i) =>
                  cell.kind === 'omit' ? (
                    <div
                      key={`omit-${i}`}
                      className={`flex shrink-0 items-center justify-center text-[10px] font-bold ${
                        isCurrent ? 'bg-accent text-onaccent' : 'bg-line text-muted'
                      }`}
                      style={{ width: 56 }}
                    >
                      {t('treeOmit', { count: cell.count })}
                    </div>
                  ) : (
                    <div
                      key={i}
                      className={`flex shrink-0 items-center justify-center border-l border-line px-1 font-mono text-[11px] first:border-l-0 ${
                        !isCurrent
                          ? 'bg-bg/60 text-muted'
                          : cell.isNew
                            ? 'bg-orange-50 font-bold text-accent2'
                            : 'bg-bg text-ink'
                      }`}
                      style={{ width: CELL_W }}
                      title={cell.text}
                    >
                      <span className="truncate">{cell.text}</span>
                    </div>
                  ),
                )}
              </div>
            </div>
          )
        })
      )}
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
  currentTable,
}: {
  tree: TreeSnapshot
  newPKs: Set<unknown>
  currentTable: string
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
          <LeafBox key={n.pageId} node={n} currentTable={currentTable} />
        ) : (
          <InternalBox key={n.pageId} node={n} isRoot={n.pageId === tree.rootPageId} />
        ),
      )}
    </div>
  )
}
