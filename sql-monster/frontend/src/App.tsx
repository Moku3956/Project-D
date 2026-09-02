import { BattleScreen } from './features/battle/BattleScreen'
import { HomePath } from './features/home-path/HomePath'
import { MonsterPreview } from './features/preview/MonsterPreview'
import { Header } from './shared/Header'
import { useGame } from './shared/store'

export default function App() {
  const screen = useGame((s) => s.screen)
  const error = useGame((s) => s.error)

  // ホーム画面はモンスターのパスを縦に辿る画面なので、スクロールする前提。
  // バトル画面はheader-barなしの1画面構成(docs/battle_screen_ui.md参照)。
  if (screen === 'home') {
    return (
      <div className="min-h-full">
        <Header />
        <HomePath />
        <MonsterPreview />
        {error && (
          <p className="mx-6 mb-6 rounded border border-danger/50 bg-danger/10 p-3 text-xs text-danger">
            {error}
          </p>
        )}
      </div>
    )
  }

  // 実ブラウザだと縦にはみ出すことがあるため、縦横同じ比率で少しだけ縮小して収める。
  // width/heightをscaleの逆数(1/0.9)にしておくことで、縮小後にちょうど親いっぱいに広がる。
  const SCALE = 0.8
  return (
    <div className="h-screen overflow-hidden">
      <div
        className="flex flex-col"
        style={{
          transform: `scale(${SCALE})`,
          transformOrigin: 'top left',
          width: `${100 / SCALE}%`,
          height: `${100 / SCALE}%`,
        }}
      >
        <BattleScreen />
      </div>
    </div>
  )
}
