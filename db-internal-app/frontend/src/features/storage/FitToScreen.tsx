import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'

const PADDING = 32

/** 子要素(ツリー図)の自然な大きさと、表示できる領域の大きさを比べて、
 * 収まるように自動で縮小率を決める。手でホイール操作やドラッグをしなくても
 * 常に全体が見える、というのが目的(拡大縮小・パンの手動操作はしない)。
 * 縮小のみ行い、領域より小さい場合に無理に拡大はしない。 */
export function FitToScreen({ children }: { children: ReactNode }) {
  const containerRef = useRef<HTMLDivElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)
  const [scale, setScale] = useState(1)

  useLayoutEffect(() => {
    const container = containerRef.current
    const content = contentRef.current
    if (!container || !content) return

    function recompute() {
      if (!container || !content) return
      const availW = container.clientWidth - PADDING * 2
      const availH = container.clientHeight - PADDING * 2
      // transform: scale() はレイアウト計算(scrollWidth/Height)には影響しないため、
      // 現在の縮小率に関係なく常にコンテンツの自然な大きさが取れる。
      const contentW = content.scrollWidth
      const contentH = content.scrollHeight
      if (contentW === 0 || contentH === 0) return
      const next = Math.min(availW / contentW, availH / contentH, 1)
      setScale(next > 0 ? next : 1)
    }

    recompute()
    const ro = new ResizeObserver(recompute)
    ro.observe(container)
    ro.observe(content)
    return () => ro.disconnect()
  })

  return (
    <div ref={containerRef} className="flex h-full w-full items-center justify-center overflow-hidden">
      <div ref={contentRef} style={{ transform: `scale(${scale})`, transformOrigin: 'center' }}>
        {children}
      </div>
    </div>
  )
}
