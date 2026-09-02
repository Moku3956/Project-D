import { create } from 'zustand'
import { api } from './api'
import type { Battle, Monster, QueryResult } from './types'

type Screen = 'home' | 'battle'

type State = {
  screen: Screen
  monsters: Monster[]
  /** プレビューを開いているモンスター。nullならモーダルは閉じている。 */
  previewing: Monster | null
  battle: Battle | null
  lastResult: QueryResult | null
  busy: boolean
  error: string | null

  loadMonsters: () => Promise<void>
  openPreview: (m: Monster) => void
  closePreview: () => void
  startBattle: (monsterId: number) => Promise<void>
  runQuery: (sql: string) => Promise<void>
  advance: () => Promise<void>
  quit: () => Promise<void>
  restart: () => Promise<void>
  backToHome: () => Promise<void>
}

export const useGame = create<State>((set, get) => {
  /** API呼び出しの共通処理。busyフラグとエラー表示をまとめて面倒みる。 */
  async function run<T>(fn: () => Promise<T>): Promise<T | undefined> {
    set({ busy: true, error: null })
    try {
      return await fn()
    } catch (e) {
      set({ error: e instanceof Error ? e.message : String(e) })
      return undefined
    } finally {
      set({ busy: false })
    }
  }

  return {
    screen: 'home',
    monsters: [],
    previewing: null,
    battle: null,
    lastResult: null,
    busy: false,
    error: null,

    loadMonsters: async () => {
      const monsters = await run(() => api.monsters())
      if (monsters) set({ monsters })
    },

    openPreview: (m) => set({ previewing: m }),
    closePreview: () => set({ previewing: null }),

    startBattle: async (monsterId) => {
      const battle = await run(() => api.startBattle(monsterId))
      if (battle) set({ battle, screen: 'battle', previewing: null, lastResult: null })
    },

    runQuery: async (sql) => {
      const battle = get().battle
      if (!battle) return
      const result = await run(() => api.runQuery(battle.id, sql))
      if (result) set({ battle: result.battle, lastResult: result })
    },

    advance: async () => {
      const battle = get().battle
      if (!battle) return
      const next = await run(() => api.advance(battle.id))
      // フェーズが変わると前のフェーズの結果は文脈が違うので消す
      if (next) set({ battle: next, lastResult: null })
    },

    quit: async () => {
      const battle = get().battle
      if (!battle) return
      const next = await run(() => api.quit(battle.id))
      if (next) set({ battle: next })
    },

    restart: async () => {
      const battle = get().battle
      if (!battle) return
      const next = await run(() => api.restart(battle.id))
      if (next) set({ battle: next, lastResult: null })
    },

    backToHome: async () => {
      set({ screen: 'home', battle: null, lastResult: null })
      await get().loadMonsters()
    },
  }
})
