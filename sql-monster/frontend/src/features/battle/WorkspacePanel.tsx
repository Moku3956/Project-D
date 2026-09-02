import { useEffect, useState } from 'react'
import { useGame } from '../../shared/store'
import type { Battle } from '../../shared/types'
import { LogPanel } from './LogPanel'
import { PlayerStatus } from './PlayerStatus'
import { QueryEditor } from './QueryEditor'
import { ResultPanel } from './ResultPanel'

type Tab = 'editor' | 'result' | 'log'

/**
 * 右カラム。上に常時表示のPLAYERステータス、下にEDITOR/RESULT/LOGのタブ。
 * クエリを実行するとRESULTタブへ自動で切り替わる。
 */
export function WorkspacePanel({ battle }: { battle: Battle }) {
  const lastResult = useGame((s) => s.lastResult)
  const [tab, setTab] = useState<Tab>('editor')

  // フェーズが変わったらエディタに戻す
  useEffect(() => {
    setTab('editor')
  }, [battle.phase])

  // クエリ実行結果(SELECTの行)が返ってきたらRESULTへ自動で切り替える
  useEffect(() => {
    if (lastResult?.columns) setTab('result')
  }, [lastResult])

  return (
    <section className="flex min-h-0 flex-[2] flex-col gap-3">
      <PlayerStatus battle={battle} />

      <div className="flex shrink-0 gap-2">
        {(['editor', 'result', 'log'] as const).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(t)}
            className={`rounded-md border px-4 py-2 font-display text-[11px] font-bold uppercase ${
              tab === t ? 'border-neon bg-neon/15 text-neon' : 'border-line bg-deep text-muted'
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      <div className="min-h-0 flex-1 rounded-xl border border-line bg-panel p-4">
        {tab === 'editor' && <QueryEditor />}
        {tab === 'result' && <ResultPanel phase={battle.phase} result={lastResult} />}
        {tab === 'log' && <LogPanel logs={battle.logs} />}
      </div>
    </section>
  )
}
