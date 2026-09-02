package btree

import (
	"fmt"

	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/types"
)

// TreeSnapshot はB+Treeの可視化用スナップショット。db-internal-appがクエリ実行
// 前後に取得し、差分表示に使う(db-internal-app/docs/spec.md「Storage」参照)。
type TreeSnapshot struct {
	RootPageID uint32
	Pages      map[uint32]PageSnapshot
}

// PageSnapshot は1ページ分の可視化用データ。スロット配列・セルの生バイト列などの
// 内部フォーマットは含めない(KVが分かれば十分という方針)。
type PageSnapshot struct {
	PageID         uint32
	IsLeaf         bool
	Keys           []string    // 内部ノードのみ。複合キーを人間可読な形にデコードしたもの
	ChildPageIDs   []uint32    // 内部ノードのみ。Keys[i]はChildPageIDs[i]の左を担当する(左子規約)
	RightmostChild uint32      // 内部ノードのみ。どのキーとも組まない最後の子
	Rows           []types.Row // 葉ノードのみ。KVそのもの
	NextLeafID     uint32      // 葉ノードのみ。範囲スキャン用の次の葉へのポインタ
}

// DumpTree はRootPageIDからツリー全体を読み取り専用で辿り、可視化用にシリアライズ
// する。既存のSearch/Insert/Delete/Scanには一切手を入れず、新規の読み取り経路として
// 追加する(db-internal-app/docs/spec.md「ストレージ可視化は既存コードへの侵襲的な
// 変更を避ける」参照)。
//
// 1つのB+Treeファイルは複数テーブルのセルを同じキー空間・同じページに混在させて
// 保持しうるため(storage/btree/docs/spec.md「キーフォーマット」参照)、葉ノードの
// セルはschema.TableIDに一致するものだけをRowsに含める。一致しないセルは無視する
// (Scan()と同じ判定方法)。
//
// バッファプール経由で読む(page.DiskManagerを直接読まない)。No-Force方式のため、
// コミット済みでもディスクにはまだ反映されず、バッファプール上のdirtyページだけが
// 最新の場合がある。
func (bt *BTree) DumpTree(schema *types.Schema) (*TreeSnapshot, error) {
	snapshot := &TreeSnapshot{
		RootPageID: bt.disk.RootPageID(),
		Pages:      make(map[uint32]PageSnapshot),
	}
	if snapshot.RootPageID == page.NoPageID {
		return snapshot, nil
	}
	if err := bt.dumpPage(snapshot.RootPageID, schema, snapshot, make(map[uint32]bool)); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (bt *BTree) dumpPage(pageID uint32, schema *types.Schema, snapshot *TreeSnapshot, visited map[uint32]bool) error {
	if visited[pageID] {
		return nil
	}
	visited[pageID] = true

	p, err := bt.bp.FetchPage(pageID)
	if err != nil {
		return err
	}
	defer bt.releasePage(p)

	n := int(p.CellCount())
	ps := PageSnapshot{PageID: pageID, IsLeaf: p.Type() == page.TypeLeaf}

	if ps.IsLeaf {
		rows := make([]types.Row, 0, n)
		for i := 0; i < n; i++ {
			cell := p.CellAt(i)
			if cellTableID(cell) != schema.TableID {
				continue
			}
			_, _, row := decodeLeafCell(cell, schema)
			rows = append(rows, row)
		}
		ps.Rows = rows
		ps.NextLeafID = nextLeafID(p)
		snapshot.Pages[pageID] = ps
		return nil
	}

	keys := make([]string, 0, n)
	children := make([]uint32, 0, n)
	for i := 0; i < n; i++ {
		cell := p.CellAt(i)
		_, key, childID := decodeInternalCell(cell)
		keys = append(keys, formatValue(key))
		children = append(children, childID)
	}
	ps.Keys = keys
	ps.ChildPageIDs = children
	ps.RightmostChild = p.RightmostChild()
	snapshot.Pages[pageID] = ps

	for _, childID := range children {
		if err := bt.dumpPage(childID, schema, snapshot, visited); err != nil {
			return err
		}
	}
	if ps.RightmostChild != page.NoPageID {
		if err := bt.dumpPage(ps.RightmostChild, schema, snapshot, visited); err != nil {
			return err
		}
	}
	return nil
}

func formatValue(v types.Value) string {
	switch val := v.(type) {
	case types.IntValue:
		return fmt.Sprintf("%d", val.V)
	case types.StringValue:
		return val.V
	case types.BoolValue:
		return fmt.Sprintf("%t", val.V)
	default:
		return ""
	}
}
