import { useState } from 'react'
import { useDbInternal } from '../../shared/store'
import { useI18n } from '../../shared/i18n'
import { DataTable } from '../storage/DataTable'
import { stripPadding } from '../../shared/displayValue'
import type { WalRecord } from '../../shared/types'

type TxnGroup = {
  txnId: number
  /** INSERT/UPDATE/DELETEのレコード(LSN昇順)。 */
  records: WalRecord[]
  /** そのtxnのCOMMIT/ABORTレコード。まだ確定していない場合はundefined。 */
  footer?: WalRecord
}

/** 元々LSN昇順のフラットな配列を、TxnIDの初出順を保ったままグルーピングする。 */
function groupByTxn(records: WalRecord[]): TxnGroup[] {
  const order: number[] = []
  const groups = new Map<number, TxnGroup>()
  for (const r of records) {
    let g = groups.get(r.txnId)
    if (!g) {
      g = { txnId: r.txnId, records: [] }
      groups.set(r.txnId, g)
      order.push(r.txnId)
    }
    if (r.op === 'COMMIT' || r.op === 'ABORT') {
      g.footer = r
    } else {
      g.records.push(r)
    }
  }
  return order.map((id) => groups.get(id)!)
}

const OP_BADGE_STYLE: Record<string, string> = {
  INSERT: 'bg-green-100 text-green-700',
  UPDATE: 'bg-blue-100 text-blue-700',
  DELETE: 'bg-red-100 text-red-700',
}

/** 展開した中身の背景色を、変わり方(追加/削除/変更)に応じてバッジと揃える。 */
const CHANGE_KIND_TINT: Record<string, string> = {
  added: 'bg-accent/10',
  removed: 'bg-red-50',
  changed: 'bg-blue-50',
}

/** SQLを実行して、左に今のテーブル、右にそのSQLで追記されたWALレコードを
 * txnごとにグルーピングして表示する。1レコード=1ページ変更という粒度と、
 * COMMIT/ABORTでトランザクションが区切られることを、説明文ではなくバッジと
 * 色・グルーピングだけで見せる(詳細な設計はdb-internal-app/docs/spec.md参照)。 */
export function WalCard() {
  const tree = useDbInternal((s) => s.tree)
  const currentTable = useDbInternal((s) => s.currentTable)
  const tables = useDbInternal((s) => s.tables)
  const walRecords = useDbInternal((s) => s.walRecords)
  const [expandedLsns, setExpandedLsns] = useState<Set<number>>(new Set())
  const { t } = useI18n()

  const columns = tables.find((tbl) => tbl.name === currentTable)?.columns ?? []
  // SELECTだけのトランザクションもCOMMITレコード自体は残るが、ページ変更を
  // 伴わないため中身が空のグループになる。書き込みの見え方を追うための画面
  // なので、そのようなグループはノイズとして除外する。
  const groups = groupByTxn(walRecords).filter((g) => g.records.length > 0)

  function toggle(lsn: number) {
    setExpandedLsns((prev) => {
      const next = new Set(prev)
      if (next.has(lsn)) next.delete(lsn)
      else next.add(lsn)
      return next
    })
  }

  return (
    <div className="rounded-2xl border border-line bg-surface p-7 shadow-sm">
      <h2 className="text-base font-bold text-ink">{t('walCardTitle')}</h2>

      <div className="mt-5 flex flex-col gap-7 lg:flex-row">
        <div className="flex flex-col gap-2.5 lg:w-[380px] lg:shrink-0">
          <h3 className="text-xs font-bold text-muted">{t('tabTable')}</h3>
          {!tree ? (
            <p className="text-sm text-muted">{t('loading')}</p>
          ) : (
            <DataTable tree={tree} columns={columns} table={currentTable} />
          )}
        </div>

        <div className="flex min-w-0 flex-1 flex-col gap-2.5">
          <h3 className="text-xs font-bold text-muted">{t('walColumnLabel')}</h3>
          {groups.length === 0 ? (
            <p className="text-sm text-muted">{t('walEmpty')}</p>
          ) : (
            <div className="flex flex-col gap-3">
              {groups.map((g) => (
                <div key={g.txnId} className="overflow-hidden rounded-xl border border-line">
                  <div className="bg-bg px-3.5 py-2 text-xs font-bold text-muted">
                    {t('walTxnLabel', { id: g.txnId })}
                  </div>

                  {g.records.map((r, i) => {
                    const canExpand = Boolean(r.changeKind && r.row)
                    const isExpanded = expandedLsns.has(r.lsn)
                    return (
                      <div
                        key={r.lsn}
                        className={`px-3.5 py-2.5 ${i < g.records.length - 1 || g.footer ? 'border-b border-line' : ''}`}
                      >
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="rounded-full bg-bg px-2 py-0.5 font-mono text-[10px] text-muted">
                            #{r.lsn}
                          </span>
                          <span
                            className={`rounded px-2 py-0.5 text-[10px] font-bold ${OP_BADGE_STYLE[r.op] ?? 'bg-bg text-muted'}`}
                          >
                            {r.op}
                          </span>
                          <span className="text-[11px] text-muted">{t('walPageLabel', { id: r.pageId })}</span>
                          {canExpand && (
                            <button
                              type="button"
                              onClick={() => toggle(r.lsn)}
                              className="ml-auto text-[11px] font-bold text-accent hover:underline"
                            >
                              {isExpanded ? '▾' : '▸'}
                            </button>
                          )}
                        </div>
                        {canExpand && isExpanded && (
                          <div
                            className={`mt-2 flex gap-4 rounded-lg border border-line px-3 py-1.5 font-mono text-xs text-ink ${CHANGE_KIND_TINT[r.changeKind!] ?? 'bg-bg'}`}
                          >
                            {r.row!.map((v, j) => (
                              <span key={j}>{stripPadding(v)}</span>
                            ))}
                          </div>
                        )}
                      </div>
                    )
                  })}

                  {g.footer && (
                    <div
                      className={`flex items-center gap-2 px-3.5 py-2 text-xs font-bold ${
                        g.footer.op === 'COMMIT' ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'
                      }`}
                    >
                      <span className="rounded-full bg-white/60 px-2 py-0.5 font-mono text-[10px]">
                        #{g.footer.lsn}
                      </span>
                      <span>{g.footer.op === 'COMMIT' ? t('walCommitLabel') : t('walAbortLabel')}</span>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
