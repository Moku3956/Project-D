/** shared/store.tsのpaddedSeedId()が作る「ゼロ埋め数値+xを連ねたパディング」を
 * 検出したら、パディング部分を無視して素の数値だけを見せる。それ以外の値は
 * そのまま返す。B+Treeのツリー図(layout.ts)・テーブル表示(DataTable.tsx)・
 * エディターのSQLプレビュー(ControlsBar.tsx)で使う共通の表示用整形。 */
export function stripPadding(v: unknown): string {
  const s = String(v)
  const m = /^(\d+)x{10,}$/.exec(s)
  return m ? String(Number(m[1])) : s
}
