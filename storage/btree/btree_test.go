package btree

import (
	"os"
	"testing"

	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/types"
)

const testTableID = uint32(1)

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
	path := t.TempDir() + "/test.db"
	dm, err := page.NewDiskManager(path)
	if err != nil {
		t.Fatal(err)
	}
	bt, err := NewBTree(dm)
	if err != nil {
		t.Fatal(err)
	}
	return bt, func() {
		if err := dm.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
}

// ---- 正常系 ----

func TestInsertAndSearch(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row, schema); err != nil {
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

func TestInsertMultipleAndScan(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	for i := int64(1); i <= 5; i++ {
		row := types.Row{Values: []types.Value{types.IntValue{V: i}, types.StringValue{V: "user"}}}
		if err := bt.Insert(testTableID, types.IntValue{V: i}, row, schema); err != nil {
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

func TestDelete(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row, schema); err != nil {
		t.Fatal(err)
	}
	if err := bt.Delete(testTableID, types.IntValue{V: 1}); err != nil {
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

	if err := bt.Insert(1, types.IntValue{V: 1}, row1, schema1); err != nil {
		t.Fatal(err)
	}
	if err := bt.Insert(2, types.IntValue{V: 1}, row2, schema2); err != nil {
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
		t.Errorf("table1 Scan件数 = %d, want 1", len(rows1))
	}
	if len(rows2) != 1 {
		t.Errorf("table2 Scan件数 = %d, want 1", len(rows2))
	}
}

// ---- 異常系 ----

func TestInsertDuplicateKey(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row, schema); err != nil {
		t.Fatal(err)
	}

	dup := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Bob"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, dup, schema); err != nil {
		t.Fatal(err)
	}

	rows, err := bt.Scan(testTableID, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Errorf("重複挿入後の件数 = %d, want 2", len(rows))
	}
}

func TestDeleteThenReinsert(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row, schema); err != nil {
		t.Fatal(err)
	}
	if err := bt.Delete(testTableID, types.IntValue{V: 1}); err != nil {
		t.Fatal(err)
	}

	row2 := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Bob"}}}
	if err := bt.Insert(testTableID, types.IntValue{V: 1}, row2, schema); err != nil {
		t.Fatal(err)
	}

	got, err := bt.Search(testTableID, types.IntValue{V: 1}, schema)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("再挿入後にnilが返った")
	}
	if got.Values[1].(types.StringValue).V != "Bob" {
		t.Errorf("name = %v, want Bob", got.Values[1])
	}
}

func TestSplitAndScanOrder(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()
	schema := testSchema()

	const n = 100
	for i := int64(1); i <= n; i++ {
		row := types.Row{Values: []types.Value{types.IntValue{V: i}, types.StringValue{V: "user"}}}
		if err := bt.Insert(testTableID, types.IntValue{V: i}, row, schema); err != nil {
			t.Fatalf("Insert(%d): %v", i, err)
		}
	}

	rows, err := bt.Scan(testTableID, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != n {
		t.Errorf("件数 = %d, want %d", len(rows), n)
	}
	for i, row := range rows {
		id := row.Values[0].(types.IntValue).V
		if id != int64(i+1) {
			t.Errorf("rows[%d].id = %d, want %d", i, id, i+1)
			break
		}
	}
}

func TestDeleteNotFound(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()

	err := bt.Delete(testTableID, types.IntValue{V: 99})
	if err == nil {
		t.Error("存在しないキーの削除でエラーが返らなかった")
	}
}
