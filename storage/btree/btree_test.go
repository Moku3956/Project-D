package btree

import (
	"os"
	"testing"

	"github.com/Moku3956/Project-D/storage/buffer"
	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
	"github.com/Moku3956/Project-D/types"
)

const testTableID = uint32(1)
const testTxnID = uint64(1)

func testSchema() *types.Schema {
	return &types.Schema{
		TableName: "users",
		TableID:   testTableID,
		Columns: []types.Column{
			{Name: "id", Type: types.DataType{Kind: types.KindIntType}, PrimaryKey: true, NotNull: true},
			{Name: "name", Type: types.DataType{Kind: types.KindVarcharType, Length: 50}, NotNull: true},
		},
	}
}

func setupBTree(t *testing.T) (*BTree, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	dm, err := page.NewDiskManager(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	wm, err := wal.NewWALManager(dir + "/test.wal")
	if err != nil {
		t.Fatal(err)
	}
	bp := buffer.NewBufferPool(dm, wm, 100)
	bt, err := NewBTree(dm, bp, wm)
	if err != nil {
		t.Fatal(err)
	}
	return bt, func() {
		if err := wm.Close(); err != nil {
			t.Fatal(err)
		}
		if err := dm.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(dbPath); err != nil {
			t.Fatal(err)
		}
	}
}

// ---- Happy path ----

// TestInsertAndSearch inserts a row and checks that Search returns it.
func TestInsertAndSearch(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row, schema, testTxnID); err != nil {
		t.Fatal(err)
	}

	got, err := bt.Search(testTableID, types.IntValue{V: 1}, schema)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected row, got nil")
	}

	name := got.Values[1].(types.StringValue).V
	if name != "Alice" {
		t.Errorf("expected Alice, got %s", name)
	}
}

// TestSearchNotFound checks that searching for a missing key returns nil, not an error.
func TestSearchNotFound(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	got, err := bt.Search(testTableID, types.IntValue{V: 99}, schema)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected nil, got row")
	}
}

// TestInsertMultipleAndScan inserts several rows and checks that Scan returns all of them.
func TestInsertMultipleAndScan(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	for i := int64(1); i <= 5; i++ {
		row := types.Row{Values: []types.Value{types.IntValue{V: i}, types.StringValue{V: "user"}}}
		if err := bt.Insert(testTableID, types.IntValue{V: i}, row, schema, testTxnID); err != nil {
			t.Fatal(err)
		}
	}

	rows, err := bt.Scan(testTableID, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Errorf("expected 5 rows, got %d", len(rows))
	}
}

// TestDelete deletes a row and checks that Search no longer finds it.
func TestDelete(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row, schema, testTxnID); err != nil {
		t.Fatal(err)
	}
	if err := bt.Delete(testTableID, types.IntValue{V: 1}, testTxnID); err != nil {
		t.Fatal(err)
	}

	got, err := bt.Search(testTableID, types.IntValue{V: 1}, schema)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected nil after delete, got row")
	}
}

func TestUpdateReplacesRow(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row, schema, testTxnID); err != nil {
		t.Fatal(err)
	}

	newRow := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alicia"}}}
	if err := bt.Update(testTableID, types.IntValue{V: 1}, newRow, schema, testTxnID); err != nil {
		t.Fatal(err)
	}

	got, err := bt.Search(testTableID, types.IntValue{V: 1}, schema)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected row, got nil")
	}
	if name := got.Values[1].(types.StringValue).V; name != "Alicia" {
		t.Errorf("name = %q, want %q", name, "Alicia")
	}
}

func TestUpdateNotFound(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	row := types.Row{Values: []types.Value{types.IntValue{V: 99}, types.StringValue{V: "Nobody"}}}
	if err := bt.Update(testTableID, types.IntValue{V: 99}, row, schema, testTxnID); err == nil {
		t.Error("expected error for missing key, got nil")
	}
}

// splitで複数の葉ページに分かれた木でも、Updateが正しい葉まで辿り着いて
// 対象の行だけを書き換えられることを確認する(updateIntoInternalの経路)。
func TestUpdateAcrossSplitTree(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	const n = 100
	for i := int64(1); i <= n; i++ {
		row := types.Row{Values: []types.Value{types.IntValue{V: i}, types.StringValue{V: "user"}}}
		if err := bt.Insert(testTableID, types.IntValue{V: i}, row, schema, testTxnID); err != nil {
			t.Fatalf("Insert(%d): %v", i, err)
		}
	}

	newRow := types.Row{Values: []types.Value{types.IntValue{V: 50}, types.StringValue{V: "updated"}}}
	if err := bt.Update(testTableID, types.IntValue{V: 50}, newRow, schema, testTxnID); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := bt.Search(testTableID, types.IntValue{V: 50}, schema)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected row, got nil")
	}
	if name := got.Values[1].(types.StringValue).V; name != "updated" {
		t.Errorf("name = %q, want %q", name, "updated")
	}

	// 他の行は影響を受けていないこと。
	rows, err := bt.Scan(testTableID, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != n {
		t.Errorf("件数 = %d, want %d", len(rows), n)
	}
}

// TestMultipleTablesIsolated checks that rows from different tables sharing one
// physical tree don't leak into each other's Search/Scan results.
func TestMultipleTablesIsolated(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()

	schema1 := &types.Schema{
		TableName: "table1",
		TableID:   1,
		Columns: []types.Column{
			{Name: "id", Type: types.DataType{Kind: types.KindIntType}, PrimaryKey: true},
			{Name: "val", Type: types.DataType{Kind: types.KindVarcharType, Length: 50}},
		},
	}
	schema2 := &types.Schema{
		TableName: "table2",
		TableID:   2,
		Columns: []types.Column{
			{Name: "id", Type: types.DataType{Kind: types.KindIntType}, PrimaryKey: true},
			{Name: "val", Type: types.DataType{Kind: types.KindVarcharType, Length: 50}},
		},
	}

	row1 := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "from-table1"}}}
	row2 := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "from-table2"}}}

	if err := bt.Insert(1, types.IntValue{V: 1}, row1, schema1, testTxnID); err != nil {
		t.Fatal(err)
	}
	if err := bt.Insert(2, types.IntValue{V: 1}, row2, schema2, testTxnID); err != nil {
		t.Fatal(err)
	}

	got1, err := bt.Search(1, types.IntValue{V: 1}, schema1)
	if err != nil {
		t.Fatal(err)
	}
	got2, err := bt.Search(2, types.IntValue{V: 1}, schema2)
	if err != nil {
		t.Fatal(err)
	}

	if got1.Values[1].(types.StringValue).V != "from-table1" {
		t.Errorf("table1 val = %v, want from-table1", got1.Values[1])
	}
	if got2.Values[1].(types.StringValue).V != "from-table2" {
		t.Errorf("table2 val = %v, want from-table2", got2.Values[1])
	}

	rows1, err := bt.Scan(1, schema1)
	if err != nil {
		t.Fatal(err)
	}
	rows2, err := bt.Scan(2, schema2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows1) != 1 {
		t.Errorf("table1 Scan count = %d, want 1", len(rows1))
	}
	if len(rows2) != 1 {
		t.Errorf("table2 Scan count = %d, want 1", len(rows2))
	}
}

// ---- Error cases ----

// TestInsertDuplicateKey checks that inserting the same key twice does not error
// and both rows persist (uniqueness is not enforced yet).
func TestInsertDuplicateKey(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row, schema, testTxnID); err != nil {
		t.Fatal(err)
	}

	dup := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Bob"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, dup, schema, testTxnID); err != nil {
		t.Fatal(err)
	}

	rows, err := bt.Scan(testTableID, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Errorf("count after duplicate insert = %d, want 2", len(rows))
	}
}

// TestDeleteThenReinsert checks that re-inserting a key after deleting it succeeds
// and returns the new value.
func TestDeleteThenReinsert(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row, schema, testTxnID); err != nil {
		t.Fatal(err)
	}
	if err := bt.Delete(testTableID, types.IntValue{V: 1}, testTxnID); err != nil {
		t.Fatal(err)
	}

	row2 := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Bob"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row2, schema, testTxnID); err != nil {
		t.Fatal(err)
	}

	got, err := bt.Search(testTableID, types.IntValue{V: 1}, schema)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("got nil after re-insert")
	}
	if got.Values[1].(types.StringValue).V != "Bob" {
		t.Errorf("name = %v, want Bob", got.Values[1])
	}
}

// TestSplitAndScanOrder inserts enough rows to trigger a page split and checks
// that Scan still returns every row in ascending PK order.
func TestSplitAndScanOrder(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	const n = 100
	for i := int64(1); i <= n; i++ {
		row := types.Row{Values: []types.Value{types.IntValue{V: i}, types.StringValue{V: "user"}}}
		if err := bt.Insert(testTableID, types.IntValue{V: i}, row, schema, testTxnID); err != nil {
			t.Fatalf("Insert(%d): %v", i, err)
		}
	}

	rows, err := bt.Scan(testTableID, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != n {
		t.Errorf("count = %d, want %d", len(rows), n)
	}
	for i, row := range rows {
		id := row.Values[0].(types.IntValue).V
		if id != int64(i+1) {
			t.Errorf("rows[%d].id = %d, want %d", i, id, i+1)
			break
		}
	}
}

// TestDeleteNotFound checks that deleting a missing key returns an error.
func TestDeleteNotFound(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()

	err := bt.Delete(testTableID, types.IntValue{V: 99}, testTxnID)
	if err == nil {
		t.Error("deleting a missing key did not return an error")
	}
}
