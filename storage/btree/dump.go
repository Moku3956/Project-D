package btree

import (
	"fmt"

	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/types"
)

// TreeSnapshot is a visualization snapshot of the B+Tree. db-internal-app takes
// one before and after query execution and uses them to show the diff (see
// db-internal-app/docs/spec.md "Storage").
type TreeSnapshot struct {
	RootPageID uint32
	Pages      map[uint32]PageSnapshot
}

// PageSnapshot is visualization data for a single page. It excludes internal
// formats like the slot array and raw cell bytes (the policy is that knowing
// the KVs is enough).
type PageSnapshot struct {
	PageID         uint32
	IsLeaf         bool
	Keys           []string    // internal nodes only. Composite keys decoded into a human-readable form
	ChildPageIDs   []uint32    // internal nodes only. Keys[i] is responsible for the left side of ChildPageIDs[i] (left-child convention)
	RightmostChild uint32      // internal nodes only. The one child with no key of its own
	Rows           []types.Row // leaf nodes only. The KVs themselves (across all tables; see RowTables for which table each row belongs to)
	RowTables      []string    // leaf nodes only. The table name Rows[i] belongs to (same length and order as Rows)
	NextLeafID     uint32      // leaf nodes only. Pointer to the next leaf, for range scans
}

// DumpTree walks the whole tree read-only starting from RootPageID and
// serializes it for visualization. It doesn't touch the existing
// Search/Insert/Delete/Scan at all; it's added as a new, separate read path
// (see db-internal-app/docs/spec.md "Storage visualization avoids invasive
// changes to existing code").
//
// A single B+Tree file can hold multiple tables' cells mixed together in the
// same key space and the same pages (see storage/btree/docs/spec.md "Key
// format"), so schemas takes a full tableID -> Schema map (not filtered to
// one table) so that a cell can be decoded correctly no matter which table it
// actually belongs to. Cells whose tableID isn't in schemas are ignored
// (e.g. leftover cells from a dropped table; this shouldn't normally happen).
//
// Reads go through the buffer pool (not page.DiskManager directly). Because
// of the No-Force policy, a commit doesn't guarantee the change is on disk
// yet; the buffer pool's dirty page may be the only up-to-date copy.
func (bt *BTree) DumpTree(schemas map[uint32]*types.Schema) (*TreeSnapshot, error) {
	snapshot := &TreeSnapshot{
		RootPageID: bt.disk.RootPageID(),
		Pages:      make(map[uint32]PageSnapshot),
	}
	if snapshot.RootPageID == page.NoPageID {
		return snapshot, nil
	}
	if err := bt.dumpPage(snapshot.RootPageID, schemas, snapshot, make(map[uint32]bool)); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (bt *BTree) dumpPage(pageID uint32, schemas map[uint32]*types.Schema, snapshot *TreeSnapshot, visited map[uint32]bool) error {
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
		ps.Rows, ps.RowTables = decodeLeafRows(p, schemas)
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
		if err := bt.dumpPage(childID, schemas, snapshot, visited); err != nil {
			return err
		}
	}
	if ps.RightmostChild != page.NoPageID {
		if err := bt.dumpPage(ps.RightmostChild, schemas, snapshot, visited); err != nil {
			return err
		}
	}
	return nil
}

// decodeLeafRows は葉ページpのセルのうち、schemasに含まれるtableIDのものだけを
// デコードして返す(dropページに他テーブル・削除済みテーブルのセルが残っている
// ことがあるため)。dumpPageとDecodePageBytesの共通処理。
func decodeLeafRows(p *page.Page, schemas map[uint32]*types.Schema) (rows []types.Row, rowTables []string) {
	n := int(p.CellCount())
	rows = make([]types.Row, 0, n)
	rowTables = make([]string, 0, n)
	for i := 0; i < n; i++ {
		cell := p.CellAt(i)
		schema, ok := schemas[cellTableID(cell)]
		if !ok {
			continue
		}
		_, _, row := decodeLeafCell(cell, schema)
		rows = append(rows, row)
		rowTables = append(rowTables, schema.TableName)
	}
	return rows, rowTables
}

// DecodePageBytes は1ページ分の生バイト列(WALレコードのRedoDataは常にページ
// 全体のスナップショットなので、そのバイト列)を葉の行としてデコードする。
// 既存のSearch/Insert/Delete/Scan(バッファプール経由)とは独立した読み取り専用
// パスで、db-internal-appのWAL可視化(そのレコードの時点でページに何が
// 入っていたか)のために追加した。内部ノードのページはnilを返す(WAL可視化では
// 葉ページの中身だけに関心があるため)。
func DecodePageBytes(data []byte, schemas map[uint32]*types.Schema) (rows []types.Row, rowTables []string) {
	p := page.FromBytes(data)
	if p.Type() != page.TypeLeaf {
		return nil, nil
	}
	return decodeLeafRows(p, schemas)
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
