package btree

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
)

// Checks that Recover replays only committed transactions' page changes and
// ignores uncommitted ones.
func TestRecoverAppliesOnlyCommitted(t *testing.T) {
	dir := t.TempDir()
	dm, err := page.NewDiskManager(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatalf("NewDiskManager: %v", err)
	}
	defer dm.Close() //nolint:errcheck

	// Allocate two empty pages up front (one for the committed case, one for uncommitted).
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

	// For the committed case: prepare mutated page bytes to use as RedoData.
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

	// txn=1 (committed): Insert record followed by a Commit record.
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

	// txn=2 (uncommitted): only an Insert record, no Commit record.
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

	// --- Simulate a restart after a crash by reopening the DiskManager/WALManager ---

	dm2, err := page.NewDiskManager(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatalf("NewDiskManager (reopen): %v", err)
	}
	defer dm2.Close() //nolint:errcheck

	wm2, err := wal.NewWALManager(filepath.Join(dir, "t.wal"))
	if err != nil {
		t.Fatalf("NewWALManager (reopen): %v", err)
	}
	defer wm2.Close() //nolint:errcheck

	if err := Recover(dm2, wm2); err != nil {
		t.Fatalf("Recover: %v", err)
	}

	// The committed change should be applied.
	got, err := dm2.ReadPage(committedID)
	if err != nil {
		t.Fatalf("ReadPage(committed): %v", err)
	}
	if !bytes.Equal(got.Bytes(), committedMutated.Bytes()) {
		t.Error("the committed transaction's change was not applied after Recover")
	}

	// The uncommitted change should not be applied (page stays empty).
	got2, err := dm2.ReadPage(uncommittedID)
	if err != nil {
		t.Fatalf("ReadPage(uncommitted): %v", err)
	}
	if bytes.Equal(got2.Bytes(), uncommittedMutated.Bytes()) {
		t.Error("the uncommitted transaction's change was applied after Recover, but shouldn't have been")
	}
}
