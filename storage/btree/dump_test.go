package btree

import (
	"fmt"
	"testing"

	"github.com/Moku3956/Project-D/types"
)

// TestDumpTreeEmptyRoot checks that DumpTree on a fresh tree returns a single empty leaf root.
func TestDumpTreeEmptyRoot(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()

	snap, err := bt.DumpTree(map[uint32]*types.Schema{testTableID: testSchema()})
	if err != nil {
		t.Fatalf("DumpTree error: %v", err)
	}
	if len(snap.Pages) != 1 {
		t.Fatalf("Pages count = %d, want 1", len(snap.Pages))
	}
	root, ok := snap.Pages[snap.RootPageID]
	if !ok {
		t.Fatal("root page missing from snapshot")
	}
	if !root.IsLeaf {
		t.Error("fresh root should be a leaf")
	}
	if len(root.Rows) != 0 {
		t.Errorf("Rows = %d, want 0", len(root.Rows))
	}
}

// DecodePageBytesはfinishPageが記録するRedoData(ページ全体のバイト列)をその
// まま渡された想定で、Insert後の葉ページから正しく行をデコードできることを
// 確認する。
func TestDecodePageBytesDecodesLeafRows(t *testing.T) {
	_, wm, _, bt := walRedoSetup(t)
	schema := testSchema()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row, schema, testTxnID); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if err := wm.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	records, err := wm.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("no WAL records found")
	}
	last := records[len(records)-1]

	rows, rowTables := DecodePageBytes(last.RedoData, map[uint32]*types.Schema{testTableID: schema})
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if name := rows[0].Values[1].(types.StringValue).V; name != "Alice" {
		t.Errorf("name = %q, want Alice", name)
	}
	if rowTables[0] != schema.TableName {
		t.Errorf("rowTables[0] = %q, want %q", rowTables[0], schema.TableName)
	}
}

// 内部ノードのページを渡した場合はnilを返す(WAL可視化では葉ページの中身だけに
// 関心があるため)。
func TestDecodePageBytesInternalPageReturnsNil(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	const n = 200
	for i := int64(1); i <= n; i++ {
		row := types.Row{Values: []types.Value{types.IntValue{V: i}, types.StringValue{V: "user"}}}
		if err := bt.Insert(testTableID, types.IntValue{V: i}, row, schema, testTxnID); err != nil {
			t.Fatalf("Insert(%d): %v", i, err)
		}
	}

	snap, err := bt.DumpTree(map[uint32]*types.Schema{testTableID: schema})
	if err != nil {
		t.Fatalf("DumpTree: %v", err)
	}
	root := snap.Pages[snap.RootPageID]
	if root.IsLeaf {
		t.Fatalf("expected root to be an internal node after %d inserts", n)
	}

	p, err := bt.bp.FetchPage(snap.RootPageID)
	if err != nil {
		t.Fatalf("FetchPage: %v", err)
	}
	defer bt.releasePage(p)

	rows, rowTables := DecodePageBytes(p.Bytes(), map[uint32]*types.Schema{testTableID: schema})
	if rows != nil || rowTables != nil {
		t.Errorf("expected nil rows/rowTables for an internal page, got %v / %v", rows, rowTables)
	}
}

// TestDumpTreeReflectsUncommittedInsert checks that DumpTree sees an insert
// immediately via the buffer pool, even before any flush to disk.
func TestDumpTreeReflectsUncommittedInsert(t *testing.T) {
	// Under No-Force, a commit doesn't guarantee the change is on disk yet; the
	// buffer pool's dirty page may be the only up-to-date copy. This is a
	// regression test that DumpTree reads through bt.bp, not the disk directly.
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row, schema, testTxnID); err != nil {
		t.Fatalf("Insert error: %v", err)
	}

	snap, err := bt.DumpTree(map[uint32]*types.Schema{testTableID: schema})
	if err != nil {
		t.Fatalf("DumpTree error: %v", err)
	}
	root := snap.Pages[snap.RootPageID]
	if len(root.Rows) != 1 {
		t.Fatalf("Rows = %d, want 1 (Insert should be visible via the buffer pool even before any flush)", len(root.Rows))
	}
	if root.Rows[0].Values[1].(types.StringValue).V != "Alice" {
		t.Errorf("Rows[0] name = %v, want Alice", root.Rows[0].Values[1])
	}
}

// TestDumpTreeAfterSplitBecomesInternal inserts enough rows to trigger a split
// and checks that DumpTree reports an internal root with all rows accounted for
// across the leaves.
func TestDumpTreeAfterSplitBecomesInternal(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	const n = 300
	for i := 1; i <= n; i++ {
		row := types.Row{Values: []types.Value{
			types.IntValue{V: int64(i)},
			types.StringValue{V: fmt.Sprintf("user-%d", i)},
		}}
		if err := bt.Insert(testTableID, types.IntValue{V: int64(i)}, row, schema, testTxnID); err != nil {
			t.Fatalf("Insert(%d) error: %v", i, err)
		}
	}

	snap, err := bt.DumpTree(map[uint32]*types.Schema{testTableID: schema})
	if err != nil {
		t.Fatalf("DumpTree error: %v", err)
	}

	root, ok := snap.Pages[snap.RootPageID]
	if !ok {
		t.Fatal("root page missing from snapshot")
	}
	if root.IsLeaf {
		t.Fatal("root should have become an internal node after enough inserts to split")
	}
	if len(root.Keys) == 0 {
		t.Error("internal root should have at least one routing key")
	}
	if len(root.Keys) != len(root.ChildPageIDs) {
		t.Errorf("Keys/ChildPageIDs length mismatch: %d vs %d", len(root.Keys), len(root.ChildPageIDs))
	}

	// Summing Rows across all leaf pages should equal the number of inserts.
	total := 0
	leafCount := 0
	for _, ps := range snap.Pages {
		if ps.IsLeaf {
			leafCount++
			total += len(ps.Rows)
		}
	}
	if leafCount < 2 {
		t.Fatalf("leafCount = %d, want >= 2 (should have split)", leafCount)
	}
	if total != n {
		t.Errorf("total rows across leaves = %d, want %d", total, n)
	}
}
