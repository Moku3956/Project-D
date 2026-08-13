package btree

import (
	"os"
	"testing"

	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/types"
)

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
	bt, err := New(dm, testSchema())
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
