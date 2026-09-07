export type SqlMode = 'INSERT' | 'UPDATE' | 'DELETE'
export const SQL_MODES: SqlMode[] = ['INSERT', 'UPDATE', 'DELETE']

/** モード切り替えボタン・行クリックで使うテンプレートSQLを組み立てる。
 * pkValueを渡すと、その行の実際のPK値(「Add Random」「まとめて追加」で
 * 作った行はB+Tree分岐用にパディングされた長い文字列であることがある)を
 * WHERE句にそのまま埋め込む。渡さなければ空クオートのままにしてユーザーに
 * 埋めさせる。PKは先頭カラムという、このアプリの他の便宜機能(paddedSeedId
 * 等)と同じ前提を使う。 */
export function buildTemplate(mode: SqlMode, table: string, columns: string[], pkValue?: string): string {
  const cols = columns.length > 0 ? columns : ['id', 'name']
  const pk = cols[0]
  const whereClause = `${pk} = '${pkValue ?? ''}'`
  if (mode === 'INSERT') {
    return `INSERT INTO ${table} VALUES (${cols.map(() => "''").join(', ')})`
  }
  if (mode === 'DELETE') {
    return `DELETE FROM ${table} WHERE ${whereClause}`
  }
  const rest = cols.slice(1)
  const sets = (rest.length > 0 ? rest : ['name']).map((c) => `${c} = ''`).join(', ')
  return `UPDATE ${table} SET ${sets} WHERE ${whereClause}`
}
