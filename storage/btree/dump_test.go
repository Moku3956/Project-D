package btree

import (
	"fmt"
	"testing"

	"github.com/Moku3956/Project-D/types"
)

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

func TestDumpTreeReflectsUncommittedInsert(t *testing.T) {
	// No-Forceでは、コミット後でもディスクにはまだ書かれず、バッファプール上の
	// dirtyページだけが最新の場合がある。DumpTreeがbt.bp経由で読むこと(disk直読みで
	// はないこと)を確認する回帰テスト。
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

	// 全ての葉ページのRowsを合計すると挿入件数と一致するはず
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
