import { useEffect } from 'react'
import { TopBar } from './shared/TopBar'
import { BTreeExplainer } from './shared/BTreeExplainer'
import { ControlsBar } from './shared/ControlsBar'
import { StorageCard } from './features/storage/StorageCard'
import { TableTabs } from './features/storage/TableTabs'
import { useDbInternal } from './shared/store'

function App() {
  useEffect(() => {
    void useDbInternal.getState().init()
  }, [])

  return (
    <div className="mx-auto max-w-[1200px] px-12">
      <TopBar />
      <BTreeExplainer />
      <TableTabs />
      <ControlsBar />
      <div className="pb-12">
        <StorageCard />
      </div>
    </div>
  )
}

export default App
