package btree

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
)

// Recoverは、コミット済みトランザクションのページ変更だけをディスクに再適用し、
// 未コミットのトランザクションのレコードは無視することを確認する。
func TestRecoverAppliesOnlyCommitted(t *testing.T) {
	dir := t.TempDir()
	dm, err := page.NewDiskManager(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatalf("NewDiskManager: %v", err)
	}
	defer dm.Close() //nolint:errcheck

	// 変更前の空ページを2枚用意する(committed用・uncommitted用)。
	committedPage, err := dm.AllocatePage(page.TypeLeaf)
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	uncommittedPage, err := dm.AllocatePage(page.TypeLeaf)
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	committedID := committedPage.PageID()
	uncommittedID := uncommittedPage.PageID()

	// committed用: 中身を変更したバイト列をRedoDataとして用意する。
	committedMutated := page.NewPage(page.TypeLeaf, committedID)
	committedMutated.AddCell([]byte("committed-data"))

	uncommittedMutated := page.NewPage(page.TypeLeaf, uncommittedID)
	uncommittedMutated.AddCell([]byte("uncommitted-data"))

	if err := dm.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	wm, err := wal.NewWALManager(filepath.Join(dir, "t.wal"))
	if err != nil {
		t.Fatalf("NewWALManager: %v", err)
	}

	// txn=1(コミット済み): Insertログ → Commitログ
	if _, err := wm.Append(&wal.LogRecord{
		TxnID:    1,
		PageID:   committedID,
		Op:       wal.OpInsert,
		RedoData: committedMutated.Bytes(),
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if _, err := wm.Append(&wal.LogRecord{TxnID: 1, Op: wal.OpCommit}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// txn=2(未コミット): Insertログのみ、Commitログなし
	if _, err := wm.Append(&wal.LogRecord{
		TxnID:    2,
		PageID:   uncommittedID,
		Op:       wal.OpInsert,
		RedoData: uncommittedMutated.Bytes(),
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	if err := wm.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := wm.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// --- ここでクラッシュ後の再起動を模して、DiskManager/WALManagerを開き直す ---

	dm2, err := page.NewDiskManager(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatalf("NewDiskManager (再オープン): %v", err)
	}
	defer dm2.Close() //nolint:errcheck

	wm2, err := wal.NewWALManager(filepath.Join(dir, "t.wal"))
	if err != nil {
		t.Fatalf("NewWALManager (再オープン): %v", err)
	}
	defer wm2.Close() //nolint:errcheck

	if err := Recover(dm2, wm2); err != nil {
		t.Fatalf("Recover: %v", err)
	}

	// コミット済み分は反映されている
	got, err := dm2.ReadPage(committedID)
	if err != nil {
		t.Fatalf("ReadPage(committed): %v", err)
	}
	if !bytes.Equal(got.Bytes(), committedMutated.Bytes()) {
		t.Error("コミット済みトランザクションの変更がRecover後に反映されていない")
	}

	// 未コミット分は反映されていない(空ページのまま)
	got2, err := dm2.ReadPage(uncommittedID)
	if err != nil {
		t.Fatalf("ReadPage(uncommitted): %v", err)
	}
	if bytes.Equal(got2.Bytes(), uncommittedMutated.Bytes()) {
		t.Error("未コミットトランザクションの変更がRecover後に反映されてしまっている")
	}
}
