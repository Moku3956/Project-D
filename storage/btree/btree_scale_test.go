package btree

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/Moku3956/Project-D/storage/buffer"
	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
	"github.com/Moku3956/Project-D/types"
)

const scaleTxnID = uint64(1)

// scaleSchema returns a test schema sized for tests that span page splits.
func scaleSchema() *types.Schema {
	return &types.Schema{
		TableName: "users", TableID: 1,
		Columns: []types.Column{
			{Name: "id", Type: types.DataType{Kind: types.KindIntType}, PrimaryKey: true},
			{Name: "name", Type: types.DataType{Kind: types.KindVarcharType, Length: 50}},
		},
	}
}

// newScaleTree builds a test B+Tree in a temporary directory.
func newScaleTree(t *testing.T) *BTree {
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
	bp := buffer.NewBufferPool(dm, wm, 200)
	bt, err := NewBTree(dm, bp, wm)
	if err != nil {
		t.Fatalf("NewBTree: %v", err)
	}
	return bt
}

// insertN inserts PKs 0..n-1 in the given order.
func insertN(t *testing.T, bt *BTree, sc *types.Schema, ids []int) {
	t.Helper()
	for _, v := range ids {
		id := int64(v)
		row := types.Row{Values: []types.Value{types.IntValue{V: id}, types.StringValue{V: "u"}}}
		if err := bt.Insert(1, types.IntValue{V: id}, row, sc, scaleTxnID); err != nil {
			t.Fatalf("Insert(%d): %v", id, err)
		}
	}
}

// assertAllReachable checks that every key 0..n-1 can be found via Search, and
// that Scan returns all of them in ascending PK order.
func assertAllReachable(t *testing.T, bt *BTree, sc *types.Schema, n int) {
	t.Helper()

	missing := 0
	for i := 0; i < n; i++ {
		r, err := bt.Search(1, types.IntValue{V: int64(i)}, sc)
		if err != nil {
			t.Fatalf("Search(%d): %v", i, err)
		}
		if r == nil {
			missing++
		}
	}
	if missing != 0 {
		t.Errorf("keys not found via Search = %d / %d", missing, n)
	}

	rows, err := bt.Scan(1, sc)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(rows) != n {
		t.Errorf("Scan count = %d, want %d", len(rows), n)
		return
	}
	for i, r := range rows {
		if got := r.Values[0].(types.IntValue).V; got != int64(i) {
			t.Errorf("Scan[%d].id = %d, want %d (ascending PK order is broken)", i, got, i)
			return
		}
	}
}

// Inserts enough rows in ascending PK order to span multiple page splits and
// checks that every row stays reachable.
func TestScaleAscendingInsert(t *testing.T) {
	bt := newScaleTree(t)
	sc := scaleSchema()

	const n = 500
	ids := make([]int, n)
	for i := range ids {
		ids[i] = i
	}
	insertN(t, bt, sc, ids)
	assertAllReachable(t, bt, sc, n)
}

// Inserts enough rows in random order to span multiple page splits and checks
// that every row stays reachable.
func TestScaleRandomInsert(t *testing.T) {
	bt := newScaleTree(t)
	sc := scaleSchema()

	const n = 500
	insertN(t, bt, sc, rand.Perm(n))
	assertAllReachable(t, bt, sc, n)
}
