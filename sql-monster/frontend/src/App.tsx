import { BattleScreen } from './features/battle/BattleScreen'
import { HomePath } from './features/home-path/HomePath'
import { MonsterPreview } from './features/preview/MonsterPreview'
import { Header } from './shared/Header'
import { useGame } from './shared/store'

export default function App() {
  const screen = useGame((s) => s.screen)
  const error = useGame((s) => s.error)

  return (
    <div className="min-h-full">
      <Header showMenu={screen === 'battle'} />

      {screen === 'home' ? (
        <>
          <HomePath />
          <MonsterPreview />
          {error && (
            <p className="mx-6 rounded border border-danger/50 bg-danger/10 p-3 text-xs text-danger">
              {error}
            </p>
          )}
        </>
      ) : (
        <BattleScreen />
      )}
    </div>
  )
}
