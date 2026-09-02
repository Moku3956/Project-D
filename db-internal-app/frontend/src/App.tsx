import { useEffect } from 'react'
import { TopBar } from './shared/TopBar'
import { EditorPanel } from './shared/EditorPanel'
import { StorageCard } from './features/storage/StorageCard'
import { useDbInternal } from './shared/store'

function App() {
  useEffect(() => {
    void useDbInternal.getState().init()
  }, [])

  return (
    <div className="mx-auto max-w-[1200px] px-12">
      <TopBar />
      <div className="flex items-start gap-6 pb-12">
        <div className="min-w-0 flex-1">
          <StorageCard />
        </div>
        <EditorPanel />
      </div>
    </div>
  )
}

export default App
