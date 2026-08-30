package btree

import (
	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
)

// Recover はWALを読み、コミット済みトランザクションのページ変更をディスクへ再適用する(Redo)。
func Recover(disk *page.DiskManager, wm *wal.WALManager) error {
	records, err := wm.ReadAll()
	if err != nil {
		return err
	}

	committed := make(map[uint64]bool)
	for _, r := range records {
		if r.Op == wal.OpCommit {
			committed[r.TxnID] = true
		}
	}

	// RedoDataはページ全体のバイト列なので、最新のページ(LSNが最大)のみでOK
	latest := make(map[uint32]*wal.LogRecord)
	for _, r := range records {
		if r.Op != wal.OpInsert && r.Op != wal.OpUpdate && r.Op != wal.OpDelete {
			continue
		}
		if !committed[r.TxnID] {
			continue
		}
		if cur, ok := latest[r.PageID]; !ok || r.LSN > cur.LSN {
			latest[r.PageID] = r
		}
	}

	for _, r := range latest {
		if err := disk.WritePage(page.FromBytes(r.RedoData)); err != nil {
			return err
		}
	}
	return nil
}
