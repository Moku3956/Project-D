// GoのAPI(sql-monster/internal/api)が返すJSONに対応する型。

export type MonsterState = 'cleared' | 'current' | 'locked'

export type Monster = {
  id: number
  name: string
  level: number
  max_hp: number
  is_boss: boolean
  state: MonsterState
}

export type Resources = {
  analysis: number
  analysis_max: number
  attack_defense: number
  attack_defense_max: number
}

/** 1ターンの4フェーズ。spec.mdの ①攻撃用分析 → ②攻撃 → ③防御用分析 → ④防御 に対応する。 */
export type PhaseNumber = 1 | 2 | 3 | 4

export type Battle = {
  id: string
  monster: Monster
  phase: PhaseNumber
  phase_name: string
  turn: number
  player_hp: number
  player_max_hp: number
  monster_hp: number
  resources: Resources
  logs: string[]
  over: boolean
  won: boolean
}

export type QueryResult = {
  columns?: string[]
  rows?: (string | number | boolean | null)[][]
  /** プランが選んだスキャン戦略。index = PKの等値検索で精密に狙えた、の意味。 */
  scan?: 'index' | 'sequential' | 'none'
  error?: string
  battle: Battle
}

/** 分析フェーズはSELECTを何度でも撃てて、手動で次のフェーズへ進む。 */
export function isAnalysisPhase(phase: PhaseNumber): boolean {
  return phase === 1 || phase === 3
}

export const PHASE_LABELS: Record<PhaseNumber, string> = {
  1: 'Attack Analysis',
  2: 'Attack Exec',
  3: 'Defense Analysis',
  4: 'Defense Exec',
}

/** フェーズごとの実行ボタンの文言(docs/battle_screen_ui.md)。 */
export const PHASE_ACTIONS: Record<PhaseNumber, string> = {
  1: 'Run Query',
  2: 'Execute Attack',
  3: 'Run Query',
  4: 'Confirm Defense',
}

/** レベル帯ごとの色。Lv1-2=シアン / Lv3-4=オレンジ / Lv5-6=レッド。 */
export function tierColor(level: number): string {
  if (level <= 2) return 'var(--color-neon)'
  if (level <= 4) return 'var(--color-amber)'
  return 'var(--color-danger)'
}
