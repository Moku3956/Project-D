import { useRef, useState, type MouseEvent as ReactMouseEvent, type ReactNode, type WheelEvent as ReactWheelEvent } from 'react'
import type { ZoomTransform } from './zoom'

const MIN_SCALE = 0.2
const MAX_SCALE = 4

/** Figmaのキャンバスのように、ホイールで拡大縮小・ドラッグでパンできる
 * ズーム可能なキャンバス。中身(children)は素の大きさのまま渡し、
 * このコンポーネントがtransformで拡大縮小・移動を担当する。 */
export function ZoomPanCanvas({
  transform,
  onTransformChange,
  children,
}: {
  transform: ZoomTransform
  onTransformChange: (t: ZoomTransform) => void
  children: ReactNode
}) {
  const containerRef = useRef<HTMLDivElement>(null)
  const dragRef = useRef<{ startX: number; startY: number; origX: number; origY: number } | null>(null)
  const [dragging, setDragging] = useState(false)

  function onWheel(e: ReactWheelEvent<HTMLDivElement>) {
    e.preventDefault()
    const rect = containerRef.current?.getBoundingClientRect()
    const cursorX = rect ? e.clientX - rect.left : 0
    const cursorY = rect ? e.clientY - rect.top : 0

    const factor = Math.exp(-e.deltaY * 0.001)
    const nextScale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, transform.scale * factor))
    // カーソル位置を基準に拡大縮小する(そのままの座標がズーム後も同じ画面位置に留まるように)
    const nextX = cursorX - ((cursorX - transform.x) / transform.scale) * nextScale
    const nextY = cursorY - ((cursorY - transform.y) / transform.scale) * nextScale
    onTransformChange({ x: nextX, y: nextY, scale: nextScale })
  }

  function onMouseDown(e: ReactMouseEvent<HTMLDivElement>) {
    if (e.button !== 0) return
    dragRef.current = { startX: e.clientX, startY: e.clientY, origX: transform.x, origY: transform.y }
    setDragging(true)
  }

  function onMouseMove(e: ReactMouseEvent<HTMLDivElement>) {
    if (!dragRef.current) return
    const dx = e.clientX - dragRef.current.startX
    const dy = e.clientY - dragRef.current.startY
    onTransformChange({ ...transform, x: dragRef.current.origX + dx, y: dragRef.current.origY + dy })
  }

  function endDrag() {
    dragRef.current = null
    setDragging(false)
  }

  return (
    <div
      ref={containerRef}
      className={`relative h-full w-full overflow-hidden bg-bg ${dragging ? 'cursor-grabbing' : 'cursor-grab'}`}
      onWheel={onWheel}
      onMouseDown={onMouseDown}
      onMouseMove={onMouseMove}
      onMouseUp={endDrag}
      onMouseLeave={endDrag}
    >
      <div
        style={{
          transform: `translate(${transform.x}px, ${transform.y}px) scale(${transform.scale})`,
          transformOrigin: '0 0',
          width: 'fit-content',
        }}
      >
        {children}
      </div>
    </div>
  )
}
