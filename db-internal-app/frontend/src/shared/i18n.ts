import { create } from 'zustand'

export type Locale = 'ja' | 'en'

const STORAGE_KEY = 'db-internal-app:locale'

function detectDefaultLocale(): Locale {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved === 'ja' || saved === 'en') return saved
  } catch {
    // localStorageが使えない環境(プライベートモード等)では既定値にフォールバック
  }
  return navigator.language.startsWith('ja') ? 'ja' : 'en'
}

type LocaleState = {
  locale: Locale
  setLocale: (locale: Locale) => void
}

/** 現在の表示言語。ページ単位の好みなのでlocalStorageに保存する(サーバー状態
 * ではない)。 */
export const useLocaleStore = create<LocaleState>((set) => ({
  locale: detectDefaultLocale(),
  setLocale: (locale) => {
    try {
      localStorage.setItem(STORAGE_KEY, locale)
    } catch {
      // 保存に失敗しても表示切り替え自体は継続する
    }
    set({ locale })
  },
}))

type Params = Record<string, string | number>

/** {name}形式のプレースホルダーをparamsで置換する。 */
function interpolate(template: string, params?: Params): string {
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (match, key: string) =>
    key in params ? String(params[key]) : match,
  )
}

const strings = {
  appSubtitle: { ja: '実際のDBの中身', en: "What's really inside the DB" },
  reset: { ja: 'リセット', en: 'Reset' },

  editorTitle: { ja: 'エディター (INSERT, DELETE, UPDATEを自由に書けます)', en: 'Editor (Write INSERT, DELETE, UPDATE freely)' },
  run: { ja: '実行 ▸', en: 'Run ▸' },
  running: { ja: '実行中…', en: 'Running…' },
  addRandom: { ja: 'ランダム追加', en: 'Add Random' },
  addRandomTitle: {
    ja: 'idを自動採番し、実在しそうな名前を1件だけランダムでINSERTする',
    en: 'Auto-generates an id and inserts one row with a realistic random name',
  },
  bulkInsert: { ja: 'まとめて追加', en: 'Bulk Insert' },
  bulkInsertTitle: {
    ja: 'idを自動採番してランダムな行をまとめてINSERTする',
    en: 'Auto-generates ids and bulk-inserts random rows',
  },

  storageTableHeading: { ja: 'テーブル({table})', en: 'Table ({table})' },
  storageTableDesc: { ja: '見た目はただのテーブルですが…', en: "It looks like an ordinary table, but…" },
  storageTreeHeading: { ja: 'B+Tree ページ構造', en: 'B+Tree Page Structure' },
  storageTreeDesc: {
    ja: '実際はこうやって保存されています！B+Treeではデータはすべて一番下の段に格納されます！',
    en: "This is how it's actually stored. In a B+Tree, all data is stored in the bottom row!",
  },
  expand: { ja: '⤢ 拡大表示', en: '⤢ Expand' },
  tabTable: { ja: 'テーブル', en: 'Table' },
  tabTree: { ja: 'B+Tree構造', en: 'B+Tree' },
  loading: { ja: '読み込み中…', en: 'Loading…' },
  treeHeadingExpanded: { ja: 'B+Tree ページ構造(拡大表示)', en: 'B+Tree Page Structure (Expanded)' },
  close: { ja: '閉じる ✕', en: 'Close ✕' },
  zoomFit: { ja: '全体表示', en: 'Fit' },
  zoomOut: { ja: '縮小', en: 'Zoom out' },
  zoomIn: { ja: '拡大', en: 'Zoom in' },

  dataTableEmpty: { ja: 'まだ行がありません', en: 'No rows yet' },
  rowClickHint: {
    ja: 'クリックすると、この行の実際のidでUPDATE/DELETE文を作成します',
    en: 'Click to build an UPDATE/DELETE statement using this row’s real id',
  },

  treeEmpty: { ja: '(空)', en: '(empty)' },
  treeOmit: { ja: '…{count}件…', en: '…{count} more…' },
  treeOtherTableTitle: { ja: '他テーブル({table})の行', en: 'Row from another table ({table})' },

  newTable: { ja: '+ 新しいテーブル', en: '+ New table' },
  newTableTitle: {
    ja: 't1, t2, ...という名前でテーブルを作り、すぐに切り替えます。',
    en: 'Creates a table named t1, t2, ... and switches to it right away.',
  },

  deleteNoMatch: {
    ja: '一致する行がなく、削除されませんでした。行をクリックしてから実行するか、WHERE句の値を確認してください。',
    en: 'No matching row was found, so nothing was deleted. Click a row first, or check the WHERE clause value.',
  },
} satisfies Record<string, Record<Locale, string>>

export type StringKey = keyof typeof strings

/** 現在のlocaleに沿ってi18n文字列を取り出すフック。 */
export function useI18n() {
  const locale = useLocaleStore((s) => s.locale)
  const setLocale = useLocaleStore((s) => s.setLocale)
  const t = (key: StringKey, params?: Params) => interpolate(strings[key][locale], params)
  return { t, locale, setLocale }
}

/** Reactフックの外(store.ts等)からi18n文字列を取り出す非フック版。 */
export function translate(key: StringKey, params?: Params): string {
  return interpolate(strings[key][useLocaleStore.getState().locale], params)
}
