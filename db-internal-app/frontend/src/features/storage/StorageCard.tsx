import { useState } from 'react'
import { useDbInternal } from '../../shared/store'
import { useI18n } from '../../shared/i18n'
import { TreeDiagram } from './TreeDiagram'
import { FitToScreen } from './FitToScreen'
import { DataTable } from './DataTable'

type Tab = 'table' | 'tree'

export function StorageCard() {
  const tree = useDbInternal((s) => s.tree)
  const newPKs = useDbInternal((s) => s.newPKs)
  const currentTable = useDbInternal((s) => s.currentTable)
  const tables = useDbInternal((s) => s.tables)
  const [tab, setTab] = useState<Tab>('table')
  const [expanded, setExpanded] = useState(false)
  const { t } = useI18n()

  const columns = tables.find((tbl) => tbl.name === currentTable)?.columns ?? []

  return (
    <div className="rounded-2xl border border-line bg-surface p-7 shadow-sm">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-bold text-ink">
            {tab === 'table' ? t('storageTableHeading', { table: currentTable }) : t('storageTreeHeading')}
          </h2>
          <p className="pt-1 text-xs text-muted">
            {tab === 'table' ? t('storageTableDesc') : t('storageTreeDesc')}
          </p>
        </div>
        {tab === 'tree' && tree && (
          <button
            type="button"
            onClick={() => setExpanded(true)}
            className="flex items-center gap-1.5 rounded-full bg-accent px-3 py-2 text-xs font-bold text-onaccent hover:brightness-110"
          >
            {t('expand')}
          </button>
        )}
      </div>

      <div className="mt-3 flex gap-1.5 rounded-full bg-bg p-1 text-xs font-bold">
        <button
          type="button"
          onClick={() => setTab('table')}
          className={`flex-1 rounded-full py-1.5 ${tab === 'table' ? 'bg-accent text-onaccent' : 'text-muted hover:text-ink'}`}
        >
          {t('tabTable')}
        </button>
        <button
          type="button"
          onClick={() => setTab('tree')}
          className={`flex-1 rounded-full py-1.5 ${tab === 'tree' ? 'bg-accent text-onaccent' : 'text-muted hover:text-ink'}`}
        >
          {t('tabTree')}
        </button>
      </div>

      <div className={tab === 'tree' ? 'overflow-x-auto pt-3' : 'pt-3'}>
        {!tree ? (
          <p className="pt-1 text-sm text-muted">{t('loading')}</p>
        ) : tab === 'table' ? (
          <DataTable tree={tree} columns={columns} table={currentTable} />
        ) : (
          <TreeDiagram tree={tree} newPKs={newPKs} currentTable={currentTable} />
        )}
      </div>

      {expanded && tree && (
        <div className="fixed inset-0 z-50 flex flex-col bg-surface">
          <div className="flex shrink-0 items-center justify-between border-b border-line px-8 py-5">
            <h2 className="text-lg font-bold text-ink">{t('treeHeadingExpanded')}</h2>
            <button
              type="button"
              onClick={() => setExpanded(false)}
              className="rounded-full bg-accent px-3.5 py-2 text-xs font-bold text-onaccent hover:brightness-110"
            >
              {t('close')}
            </button>
          </div>
          <div className="min-h-0 flex-1">
            <FitToScreen>
              <TreeDiagram tree={tree} newPKs={newPKs} currentTable={currentTable} />
            </FitToScreen>
          </div>
        </div>
      )}
    </div>
  )
}
