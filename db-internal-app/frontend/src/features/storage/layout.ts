import type { PageSnapshot, TreeSnapshot } from '../../shared/types'
import { stripPadding } from './displayValue'

export const CELL_W = 84
export const CELL_H = 34
export const LEAF_LABEL_H = 16 // 葉の上部に表示するテーブル名ラベルの高さ
const OMIT_W = 56
const INTERNAL_W = 120
const INTERNAL_H = 44
const LEVEL_GAP = 56
const SIBLING_GAP = 24
const HEAD_COUNT = 2
const TAIL_COUNT = 4 // 先頭2件+末尾2件

export type Cell = { kind: 'row'; text: string; isNew: boolean } | { kind: 'omit'; count: number }

export type LayoutNode = {
  pageId: number
  isLeaf: boolean
  x: number
  y: number
  width: number
  height: number
  cells?: Cell[] // 葉のみ
}

export type LayoutEdge = {
  fromX: number
  fromY: number
  toX: number
  toY: number
  label?: string
}

export type Layout = { nodes: LayoutNode[]; edges: LayoutEdge[]; width: number; height: number }

const MAX_FIELD_LEN = 12

/** 1フィールドの表示を短く切り詰める。分岐を起こしやすくするためseedMany側で
 * わざと長いidを入れることがあるが、表示は常にダミーだと分かる簡潔な値にする
 * (実データの長さ・パディングの都合と見た目は別の話、というユーザー指示による)。 */
function truncateField(v: unknown): string {
  const s = stripPadding(v)
  return s.length > MAX_FIELD_LEN ? s.slice(0, MAX_FIELD_LEN) + '…' : s
}

/** rowsを「先頭2件＋省略＋末尾2件」に圧縮する(db-internal-app/docs/spec.md「Storage」参照)。
 * 全体が4件以下ならそのまま全件を返す。 */
function truncateCells(rows: unknown[][], newPKs: Set<unknown>): Cell[] {
  const toCell = (row: unknown[]): Cell => {
    const fields = row.map(truncateField)
    return {
      kind: 'row',
      text: fields.length > 1 ? `${fields[0]}: ${fields.slice(1).join(', ')}` : fields[0],
      isNew: newPKs.has(row[0]),
    }
  }
  if (rows.length <= TAIL_COUNT) {
    return rows.map(toCell)
  }
  const head = rows.slice(0, HEAD_COUNT).map(toCell)
  const tail = rows.slice(rows.length - HEAD_COUNT).map(toCell)
  const omitted = rows.length - HEAD_COUNT * 2
  return [...head, { kind: 'omit', count: omitted }, ...tail]
}

function leafWidth(cells: Cell[]): number {
  if (cells.length === 0) return CELL_W // 空リーフの「(空)」表示ぶんの最低幅
  return cells.reduce((w, c) => w + (c.kind === 'omit' ? OMIT_W : CELL_W), 0)
}

/** 内部ノードのchildren(セルのchild + RightmostChild)と、各境界のラベルを組み立てる。
 * keysはPKそのもの(パディングされた長い文字列のことがある)なので、ラベルは
 * truncateFieldで必ず短く切り詰める。 */
function childEdgesOf(page: PageSnapshot): { childID: number; label: string }[] {
  const keys = page.keys ?? []
  const children = page.childPageIds ?? []
  const out: { childID: number; label: string }[] = []
  children.forEach((childID, i) => {
    if (i === 0) {
      out.push({ childID, label: keys[0] !== undefined ? `< ${truncateField(keys[0])}` : '' })
    } else {
      out.push({ childID, label: `${truncateField(keys[i - 1])} 〜 <${truncateField(keys[i])}` })
    }
  })
  if (page.rightmostChild !== undefined) {
    const last = keys[keys.length - 1]
    out.push({ childID: page.rightmostChild, label: last !== undefined ? `>= ${truncateField(last)}` : '' })
  }
  return out
}

/** TreeSnapshotを画面座標つきのノード・エッジ一覧に変換する。 */
export function layoutTree(tree: TreeSnapshot, newPKs: Set<unknown>): Layout {
  const nodes: LayoutNode[] = []
  const edges: LayoutEdge[] = []

  // 幅を子から積み上げで計算する(post-order)
  function subtreeWidth(pageId: number): number {
    const page = tree.pages[pageId]
    if (!page) return INTERNAL_W
    if (page.isLeaf) {
      return leafWidth(truncateCells(page.rows ?? [], newPKs))
    }
    const childEdges = childEdgesOf(page)
    if (childEdges.length === 0) return INTERNAL_W
    const total = childEdges.reduce((w, e) => w + subtreeWidth(e.childID), 0)
    return Math.max(INTERNAL_W, total + SIBLING_GAP * (childEdges.length - 1))
  }

  // x/yを割り当てる(pre-order)。leftXはこのサブツリーの左端。
  function place(pageId: number, leftX: number, depth: number): number {
    const page = tree.pages[pageId]
    const y = depth * (INTERNAL_H + LEVEL_GAP)
    if (!page) {
      nodes.push({
        pageId,
        isLeaf: true,
        x: leftX + INTERNAL_W / 2,
        y,
        width: INTERNAL_W,
        height: CELL_H + LEAF_LABEL_H,
        cells: [],
      })
      return leftX + INTERNAL_W
    }

    if (page.isLeaf) {
      const cells = truncateCells(page.rows ?? [], newPKs)
      const width = Math.max(leafWidth(cells), 10)
      nodes.push({ pageId, isLeaf: true, x: leftX + width / 2, y, width, height: CELL_H + LEAF_LABEL_H, cells })
      return leftX + width
    }

    const childEdges = childEdgesOf(page)
    let cursor = leftX
    const childNodes: LayoutNode[] = []
    for (const e of childEdges) {
      const w = subtreeWidth(e.childID)
      place(e.childID, cursor, depth + 1)
      childNodes.push(nodes[nodes.length - 1]) // placeは自分のノードを最後にpushする
      cursor += w + SIBLING_GAP
    }
    const width = INTERNAL_W
    const centerX =
      childNodes.length > 0 ? (childNodes[0].x + childNodes[childNodes.length - 1].x) / 2 : leftX + width / 2
    const parentNode: LayoutNode = { pageId, isLeaf: false, x: centerX, y, width, height: INTERNAL_H }
    nodes.push(parentNode)

    childEdges.forEach((e, i) => {
      const childNode = childNodes[i]
      edges.push({
        fromX: parentNode.x,
        fromY: parentNode.y + parentNode.height,
        toX: childNode.x,
        toY: childNode.y,
        label: e.label,
      })
    })

    return Math.max(cursor - SIBLING_GAP, leftX + width)
  }

  place(tree.rootPageId, 0, 0)

  const maxX = Math.max(...nodes.map((n) => n.x + n.width / 2), 0)
  const minX = Math.min(...nodes.map((n) => n.x - n.width / 2), 0)
  const maxY = Math.max(...nodes.map((n) => n.y + n.height), 0)

  // 全体をminXぶん右にずらして0始まりにする
  const shift = -minX
  for (const n of nodes) n.x += shift
  for (const e of edges) {
    e.fromX += shift
    e.toX += shift
  }

  return { nodes, edges, width: maxX - minX, height: maxY }
}
