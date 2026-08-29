package btree

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/types"
)

// scaleSchema はページ分割をまたぐ規模のテスト用スキーマを返す。
func scaleSchema() *types.Schema {
	return &types.Schema{
		TableName: "users", TableID: 1,
		Columns: []types.Column{
			{Name: "id", Type: types.DataType{Kind: types.KindIntType}, PrimaryKey: true},
			{Name: "name", Type: types.DataType{Kind: types.KindVarcharType, Length: 50}},
		},
	}
}

// newScaleTree はテスト用のB+Treeを一時ディレクトリ上に構築する。
func newScaleTree(t *testing.T) *BTree {
	t.Helper()
	dm, err := page.NewDiskManager(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("NewDiskManager: %v", err)
	}
	t.Cleanup(func() { dm.Close() }) //nolint:errcheck
	bt, err := NewBTree(dm)
	if err != nil {
		t.Fatalf("NewBTree: %v", err)
	}
	return bt
}

// insertN は0..n-1のPKを与えられた順序で挿入する。
func insertN(t *testing.T, bt *BTree, sc *types.Schema, ids []int) {
	t.Helper()
	for _, v := range ids {
		id := int64(v)
		row := types.Row{Values: []types.Value{types.IntValue{V: id}, types.StringValue{V: "u"}}}
		if err := bt.Insert(1, types.IntValue{V: id}, row, sc); err != nil {
			t.Fatalf("Insert(%d): %v", id, err)
		}
	}
}

// assertAllReachable は0..n-1の全キーがSearchで引け、ScanがPK昇順で全件返すことを確認する。
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
		t.Errorf("Searchで引けなかった件数 = %d / %d", missing, n)
	}

	rows, err := bt.Scan(1, sc)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(rows) != n {
		t.Errorf("Scan件数 = %d, want %d", len(rows), n)
		return
	}
	for i, r := range rows {
		if got := r.Values[0].(types.IntValue).V; got != int64(i) {
			t.Errorf("Scan[%d].id = %d, want %d（PK昇順が崩れている）", i, got, i)
			return
		}
	}
}

// ページ分割をまたぐ規模で、昇順に挿入した全件が引けることを確認する。
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

// ページ分割をまたぐ規模で、ランダム順に挿入した全件が引けることを確認する。
func TestScaleRandomInsert(t *testing.T) {
	bt := newScaleTree(t)
	sc := scaleSchema()

	const n = 500
	insertN(t, bt, sc, rand.Perm(n))
	assertAllReachable(t, bt, sc, n)
}
