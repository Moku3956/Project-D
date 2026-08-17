package btree

import (
	"os"
	"testing"

	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/types"
)

// ---- 正常系 ----

func testSchema() *types.Schema {
	return &types.Schema{
		TableName: "users",
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
	bt, err := NewBTree(dm, testSchema())
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

func TestInsertAndSearch(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(types.IntValue{V: 1}, row); err != nil {
		t.Fatal(err)
	}

	got, err := bt.Search(types.IntValue{V: 1})
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

	got, err := bt.Search(types.IntValue{V: 99})
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

	for i := int64(1); i <= 5; i++ {
		row := types.Row{Values: []types.Value{types.IntValue{V: i}, types.StringValue{V: "user"}}}
		if err := bt.Insert(types.IntValue{V: i}, row); err != nil {
			t.Fatal(err)
		}
	}

	rows, err := bt.Scan()
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

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(types.IntValue{V: 1}, row); err != nil {
		t.Fatal(err)
	}
	if err := bt.Delete(types.IntValue{V: 1}); err != nil {
		t.Fatal(err)
	}

	got, err := bt.Search(types.IntValue{V: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected nil after delete, got row")
	}
}

// ---- 異常系 ----
func TestInsertDuplicateKey(t *testing.T) {
	bt, cleanup := setupBTree(t)
	defer cleanup()

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(types.IntValue{V: 1}, row); err != nil {
		t.Fatal(err)
	}

	dup := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Bob"}}}
	if err := bt.Insert(types.IntValue{V: 1}, dup); err != nil {
		t.Fatal(err)
	}

	rows, err := bt.Scan()
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

	row := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Alice"}}}
	if err := bt.Insert(types.IntValue{V: 1}, row); err != nil {
		t.Fatal(err)
	}
	if err := bt.Delete(types.IntValue{V: 1}); err != nil {
		t.Fatal(err)
	}

	row2 := types.Row{Values: []types.Value{types.IntValue{V: 1}, types.StringValue{V: "Bob"}}}
	if err := bt.Insert(types.IntValue{V: 1}, row2); err != nil {
		t.Fatal(err)
	}

	got, err := bt.Search(types.IntValue{V: 1})
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

	const n = 100
	for i := int64(1); i <= n; i++ {
		row := types.Row{Values: []types.Value{types.IntValue{V: i}, types.StringValue{V: "user"}}}
		if err := bt.Insert(types.IntValue{V: i}, row); err != nil {
			t.Fatalf("Insert(%d): %v", i, err)
		}
	}

	rows, err := bt.Scan()
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

	err := bt.Delete(types.IntValue{V: 99})
	if err == nil {
		t.Error("存在しないキーの削除でエラーが返らなかった")
	}
}
