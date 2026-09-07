import { useEffect } from 'react'
import { Routes, Route } from 'react-router-dom'
import { TopBar } from './shared/TopBar'
import { BTreeExplainer } from './shared/BTreeExplainer'
import { ControlsBar } from './shared/ControlsBar'
import { StorageCard } from './features/storage/StorageCard'
import { TableTabs } from './features/storage/TableTabs'
import { HomeCard } from './features/home/HomeCard'
import { WalCard } from './features/wal/WalCard'
import { useDbInternal } from './shared/store'

function StoragePage() {
  return (
    <>
      <BTreeExplainer />
      <TableTabs />
      <ControlsBar />
      <div className="pb-12">
        <StorageCard />
      </div>
    </>
  )
}

function WalPage() {
  return (
    <div className="pb-12">
      <WalCard />
    </div>
  )
}

function App() {
  useEffect(() => {
    void useDbInternal.getState().init()
  }, [])

  return (
    <div className="mx-auto max-w-[1200px] px-12">
      <TopBar />
      <Routes>
        <Route path="/" element={<HomeCard />} />
        <Route path="/storage" element={<StoragePage />} />
        <Route path="/wal" element={<WalPage />} />
      </Routes>
    </div>
  )
}

export default App
