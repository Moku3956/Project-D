package btree

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/Moku3956/Project-D/storage/buffer"
	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
	"github.com/Moku3956/Project-D/types"
)

// walRedoSetup はWAL・バッファプール・BTreeを一時ディレクトリ上に構築する。
func walRedoSetup(t *testing.T) (*page.DiskManager, *wal.WALManager, *buffer.BufferPool, *BTree) {
	t.Helper()
	dir := t.TempDir()
	dm, err := page.NewDiskManager(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatalf("NewDiskManager: %v", err)
	}
	t.Cleanup(func() { dm.Close() }) //nolint:errcheck
	wm, err := wal.NewWALManager(filepath.Join(dir, "t.wal"))
	if err != nil {
		t.Fatalf("NewWALManager: %v", err)
	}
	t.Cleanup(func() { wm.Close() }) //nolint:errcheck
	bp := buffer.NewBufferPool(dm, wm, 100)
	bt, err := NewBTree(dm, bp, wm)
	if err != nil {
		t.Fatalf("NewBTree: %v", err)
	}
	return dm, wm, bp, bt
}

// Insertした直後は、FlushAllを呼ぶまでディスク上のページが変化しないこと(No-Steal)を確認する。
func TestInsertNoStealBeforeFlush(t *testing.T) {
	dm, _, bp, bt := walRedoSetup(t)
	schema := testSchema()

	rootID := dm.RootPageID()
	before, err := dm.ReadPage(rootID)
	if err != nil {
		t.Fatalf("ReadPage: %v", err)
	}
	beforeBytes := append([]byte(nil), before.Bytes()...)

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row, schema, testTxnID); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	// FlushAllを呼ぶ前は、ディスク上のページはInsert前のバイト列のまま。
	afterInsert, err := dm.ReadPage(rootID)
	if err != nil {
		t.Fatalf("ReadPage: %v", err)
	}
	if !bytes.Equal(beforeBytes, afterInsert.Bytes()) {
		t.Error("FlushAll前にディスク上のページが変化した(No-Stealに違反している)")
	}

	// FlushAllを呼んで初めてディスクに反映される。
	if err := bp.FlushAll(map[uint64]bool{testTxnID: true}); err != nil {
		t.Fatalf("FlushAll: %v", err)
	}
	afterFlush, err := dm.ReadPage(rootID)
	if err != nil {
		t.Fatalf("ReadPage: %v", err)
	}
	if bytes.Equal(beforeBytes, afterFlush.Bytes()) {
		t.Error("FlushAll後もディスク上のページがInsert前のまま変化していない")
	}
}

// Insertが実際にRedoData入りのWALレコードとして記録されることを確認する。
func TestInsertLogsRedoData(t *testing.T) {
	_, wm, _, bt := walRedoSetup(t)
	schema := testSchema()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row, schema, testTxnID); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	// AppendはバッファにためるだけなのでFlushしてからファイルを読む。
	if err := wm.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	records, err := wm.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	found := false
	for _, r := range records {
		if r.Op == wal.OpInsert && r.TxnID == testTxnID && len(r.RedoData) == page.PageSize {
			found = true
			break
		}
	}
	if !found {
		t.Error("Insertを表すRedoログ(ページ全体のRedoData)が見つからない")
	}
}
