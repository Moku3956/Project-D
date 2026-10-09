import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { useI18n } from '../../shared/i18n'

const PADDING = 32
const MIN_SCALE = 0.1
const MAX_SCALE = 4
const ZOOM_STEP = 1.3

/** 子要素(ツリー図)の自然な大きさと、表示できる領域の大きさを比べて、開いた
 * 直後は全体が収まる縮小率を自動で当てる。そこから先はボタンで自由に拡大・
 * 縮小でき(「拡大表示があまりよくない。拡大できる制限があり、思うように
 * 拡大できない」というユーザー指摘により、shrink-onlyの自動フィットのみ
 * だった挙動を拡張)、拡大して画面に収まらなくなった分はブラウザ標準の
 * スクロールでパンする(以前試した独自のドラッグ/ホイールでのパン・ズームは
 * 「ちゃんと動かない」とユーザーに指摘され撤去した経緯があるため、今回は
 * 独自のポインタイベント処理を増やさず、ネイティブスクロールに任せる)。 */
export function FitToScreen({ children }: { children: ReactNode }) {
  const containerRef = useRef<HTMLDivElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)
  const [fitScale, setFitScale] = useState(1)
  const [scale, setScale] = useState(1)
  const [naturalSize, setNaturalSize] = useState({ w: 0, h: 0 })
  const initializedRef = useRef(false)
  const { t } = useI18n()

  useLayoutEffect(() => {
    const container = containerRef.current
    const content = contentRef.current
    if (!container || !content) return

    function recompute() {
      if (!container || !content) return
      const availW = container.clientWidth - PADDING * 2
      const availH = container.clientHeight - PADDING * 2
      // transform: scale()はレイアウト計算(scrollWidth/Height)に影響しないため、
      // contentRef自体には常にtransformを付けず、現在の拡大率に関係なく
      // 自然な大きさを取れるようにしている。
      const contentW = content.scrollWidth
      const contentH = content.scrollHeight
      if (contentW === 0 || contentH === 0) return
      const fit = Math.min(availW / contentW, availH / contentH, 1)
      const fitClamped = fit > 0 ? fit : 1
      setFitScale(fitClamped)
      setNaturalSize({ w: contentW, h: contentH })
      if (!initializedRef.current) {
        initializedRef.current = true
        setScale(fitClamped)
      }
    }

    recompute()
    const ro = new ResizeObserver(recompute)
    ro.observe(container)
    ro.observe(content)
    return () => ro.disconnect()
  }, [])

  const clamp = (s: number) => Math.min(MAX_SCALE, Math.max(MIN_SCALE, s))
  const zoomIn = () => setScale((s) => clamp(s * ZOOM_STEP))
  const zoomOut = () => setScale((s) => clamp(s / ZOOM_STEP))
  const resetZoom = () => setScale(fitScale)

  const scaledW = naturalSize.w * scale
  const scaledH = naturalSize.h * scale

  return (
    <div className="relative flex h-full w-full flex-col">
      <div ref={containerRef} className="flex-1 overflow-auto">
        <div className="flex" style={{ minWidth: '100%', minHeight: '100%', padding: PADDING }}>
          <div className="m-auto" style={{ width: scaledW || undefined, height: scaledH || undefined }}>
            <div
              style={{
                width: naturalSize.w || undefined,
                height: naturalSize.h || undefined,
                transform: `scale(${scale})`,
                transformOrigin: 'top left',
              }}
            >
              <div ref={contentRef}>{children}</div>
            </div>
          </div>
        </div>
      </div>

      <div className="pointer-events-none absolute inset-x-0 bottom-4 flex justify-center">
        <div className="pointer-events-auto flex items-center gap-1 rounded-full border border-line bg-surface px-2 py-1.5 shadow-lg">
          <button
            type="button"
            onClick={zoomOut}
            title={t('zoomOut')}
            className="flex h-7 w-7 items-center justify-center rounded-full text-base font-bold text-ink hover:bg-bg"
          >
            −
          </button>
          <span className="w-12 text-center text-xs font-bold text-muted">{Math.round(scale * 100)}%</span>
          <button
            type="button"
            onClick={zoomIn}
            title={t('zoomIn')}
            className="flex h-7 w-7 items-center justify-center rounded-full text-base font-bold text-ink hover:bg-bg"
          >
            ＋
          </button>
          <button
            type="button"
            onClick={resetZoom}
            title={t('zoomFit')}
            className="ml-1 rounded-full border border-line px-2.5 py-1 text-[11px] font-bold text-muted hover:text-ink"
          >
            {t('zoomFit')}
          </button>
        </div>
      </div>
    </div>
  )
}
