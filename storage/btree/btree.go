package btree

import (
	"encoding/binary"
	"fmt"

	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/types"
)

// BTree はB+Treeの操作を提供する。全テーブルで単一インスタンスを共有する。
type BTree struct {
	disk *page.DiskManager
}

func NewBTree(disk *page.DiskManager) (*BTree, error) {
	bt := &BTree{disk: disk}

	if disk.RootPageID() == page.NoPageID {
		info, err := disk.AllocatePage(page.TypeLeaf)
		if err != nil {
			return nil, err
		}
		initLeafLinks(info)
		if err := disk.WritePage(info); err != nil {
			return nil, err
		}
		if err := disk.SetRootPageID(info.PageID()); err != nil {
			return nil, err
		}
	}
	return bt, nil
}

// Search はtableIDとキーに対応するRowを返す。見つからない場合はnil。
func (bt *BTree) Search(tableID uint32, key types.Value, schema *types.Schema) (*types.Row, error) {
	leafPage, err := bt.findLeaf(tableID, key)
	if err != nil {
		return nil, err
	}
	idx, found := bt.searchInLeaf(leafPage, tableID, key)
	if !found {
		return nil, nil
	}
	cell := leafPage.CellAt(idx)
	_, _, row := decodeLeafCell(cell, schema)
	return &row, nil
}

// Insert はtableIDとキーとRowを挿入する。
func (bt *BTree) Insert(tableID uint32, key types.Value, row types.Row, schema *types.Schema) error {
	rootID := bt.disk.RootPageID()
	upTableID, upKey, newPageID, err := bt.insertRecursive(rootID, tableID, key, row, schema)
	if err != nil {
		return err
	}
	if newPageID != 0 {
		return bt.createNewRoot(rootID, upTableID, upKey, newPageID)
	}
	return nil
}

// Delete はtableIDとキーに対応するレコードを削除する。
func (bt *BTree) Delete(tableID uint32, key types.Value) error {
	leafPage, err := bt.findLeaf(tableID, key)
	if err != nil {
		return err
	}
	idx, found := bt.searchInLeaf(leafPage, tableID, key)
	if !found {
		return fmt.Errorf("key not found")
	}
	leafPage.DeleteCell(idx)
	return bt.disk.WritePage(leafPage)
}

// Scan はtableIDに属する全レコードを葉ノードのリンクリストを辿って返す。
func (bt *BTree) Scan(tableID uint32, schema *types.Schema) ([]types.Row, error) {
	var rows []types.Row
	leafID := bt.findLeftmostLeaf(bt.disk.RootPageID())
	for leafID != page.NoPageID {
		p, err := bt.disk.ReadPage(leafID)
		if err != nil {
			return nil, err
		}
		n := int(p.CellCount())
		done := false
		for i := 0; i < n; i++ {
			cell := p.CellAt(i)
			tid := cellTableID(cell)
			if tid < tableID {
				continue
			}
			if tid > tableID {
				done = true
				break
			}
			_, _, row := decodeLeafCell(cell, schema)
			rows = append(rows, row)
		}
		if done {
			break
		}
		leafID = nextLeafID(p)
	}
	return rows, nil
}

// --- 内部実装 ---

func (bt *BTree) findLeaf(tableID uint32, key types.Value) (*page.Page, error) {
	pageID := bt.disk.RootPageID()
	for {
		p, err := bt.disk.ReadPage(pageID)
		if err != nil {
			return nil, err
		}
		if p.Type() == page.TypeLeaf {
			return p, nil
		}
		pageID = bt.findChildPageID(p, tableID, key)
	}
}

func (bt *BTree) findChildPageID(p *page.Page, tableID uint32, key types.Value) uint32 {
	n := int(p.CellCount())
	for i := 0; i < n; i++ {
		cell := p.CellAt(i)
		kid, k, childID := decodeInternalCell(cell)
		if compareCompositeKeys(tableID, key, kid, k) < 0 {
			if i == 0 {
				return childID
			}
			prevCell := p.CellAt(i - 1)
			_, _, prevChild := decodeInternalCell(prevCell)
			return prevChild
		}
	}
	return p.RightmostChild()
}

func (bt *BTree) searchInLeaf(p *page.Page, tableID uint32, key types.Value) (int, bool) {
	n := int(p.CellCount())
	for i := 0; i < n; i++ {
		cell := p.CellAt(i)
		kid, k, _ := decodeCompositeKey(cell)
		cmp := compareCompositeKeys(kid, k, tableID, key)
		if cmp == 0 {
			return i, true
		}
		if cmp > 0 {
			break
		}
	}
	return 0, false
}

// insertRecursive はpageIDのサブツリーにkey/rowを挿入する。
// 分割が発生した場合は(分割tableID, 分割キー, 新しい右ページID)を返す。
func (bt *BTree) insertRecursive(pageID uint32, tableID uint32, key types.Value, row types.Row, schema *types.Schema) (uint32, types.Value, uint32, error) {
	p, err := bt.disk.ReadPage(pageID)
	if err != nil {
		return 0, nil, 0, err
	}
	if p.Type() == page.TypeLeaf {
		return bt.insertIntoLeaf(p, tableID, key, row, schema)
	}
	return bt.insertIntoInternal(p, tableID, key, row, schema)
}

// findInsertPos は複合キー順を保つ挿入位置を返す。
// 葉・内部どちらのセルも先頭が複合キーなので共通で使える。
func (bt *BTree) findInsertPos(p *page.Page, tableID uint32, key types.Value) int {
	n := int(p.CellCount())
	for i := 0; i < n; i++ {
		kid, k, _ := decodeCompositeKey(p.CellAt(i))
		if compareCompositeKeys(kid, k, tableID, key) > 0 {
			return i
		}
	}
	return n
}

func (bt *BTree) insertIntoLeaf(p *page.Page, tableID uint32, key types.Value, row types.Row, schema *types.Schema) (uint32, types.Value, uint32, error) {
	cell := encodeLeafCell(tableID, key, row, schema)
	if p.InsertCellAt(bt.findInsertPos(p, tableID, key), cell) {
		return 0, nil, 0, bt.disk.WritePage(p)
	}
	return bt.splitLeaf(p, tableID, key, row, schema)
}

func (bt *BTree) insertIntoInternal(p *page.Page, tableID uint32, key types.Value, row types.Row, schema *types.Schema) (uint32, types.Value, uint32, error) {
	childID := bt.findChildPageID(p, tableID, key)
	upTableID, upKey, newChildID, err := bt.insertRecursive(childID, tableID, key, row, schema)
	if err != nil {
		return 0, nil, 0, err
	}
	if newChildID == 0 {
		return 0, nil, 0, nil
	}
	cell := encodeInternalCell(upTableID, upKey, newChildID)
	if p.InsertCellAt(bt.findInsertPos(p, upTableID, upKey), cell) {
		return 0, nil, 0, bt.disk.WritePage(p)
	}
	return bt.splitInternal(p, upTableID, upKey, newChildID)
}

func (bt *BTree) splitLeaf(p *page.Page, tableID uint32, key types.Value, row types.Row, schema *types.Schema) (uint32, types.Value, uint32, error) {
	rightPage, err := bt.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		return 0, nil, 0, err
	}
	initLeafLinks(rightPage)

	// 分割前のpの次ページを退避する。右ページをpとその次の間に挟むため、
	// ここで取っておかないと以降の葉が連結リストから切り離される。
	oldNext := nextLeafID(p)

	n := int(p.CellCount())
	cells := make([][]byte, n)
	for i := 0; i < n; i++ {
		c := p.CellAt(i)
		cp := make([]byte, len(c))
		copy(cp, c)
		cells[i] = cp
	}

	newCell := encodeLeafCell(tableID, key, row, schema)
	cells = append(cells, newCell)
	sortCells(cells)

	mid := len(cells) / 2
	resetPage(p)
	for _, c := range cells[:mid] {
		p.AddCell(c)
	}
	for _, c := range cells[mid:] {
		rightPage.AddCell(c)
	}

	// p → rightPage → （分割前のpの次） の順に繋ぎ直す
	setNextLeafID(rightPage, oldNext)
	setNextLeafID(p, rightPage.PageID())
	setPrevLeafID(rightPage, p.PageID())

	splitTableID, splitKey, _ := decodeCompositeKey(cells[mid])

	if err := bt.disk.WritePage(p); err != nil {
		return 0, nil, 0, err
	}
	if err := bt.disk.WritePage(rightPage); err != nil {
		return 0, nil, 0, err
	}
	return splitTableID, splitKey, rightPage.PageID(), nil
}

func (bt *BTree) splitInternal(p *page.Page, tableID uint32, key types.Value, rightChildID uint32) (uint32, types.Value, uint32, error) {
	newRight, err := bt.disk.AllocatePage(page.TypeInternal)
	if err != nil {
		return 0, nil, 0, err
	}

	n := int(p.CellCount())
	cells := make([][]byte, n)
	for i := 0; i < n; i++ {
		c := p.CellAt(i)
		cp := make([]byte, len(c))
		copy(cp, c)
		cells[i] = cp
	}
	newCell := encodeInternalCell(tableID, key, rightChildID)
	cells = append(cells, newCell)
	sortCells(cells)

	mid := len(cells) / 2
	midTableID, midKey, _ := decodeCompositeKey(cells[mid])

	resetPage(p)
	for _, c := range cells[:mid] {
		p.AddCell(c)
	}
	_, _, midChild := decodeInternalCell(cells[mid])
	p.SetRightmostChild(midChild)

	for _, c := range cells[mid+1:] {
		newRight.AddCell(c)
	}
	_, _, lastChild := decodeInternalCell(cells[len(cells)-1])
	newRight.SetRightmostChild(lastChild)

	if err := bt.disk.WritePage(p); err != nil {
		return 0, nil, 0, err
	}
	if err := bt.disk.WritePage(newRight); err != nil {
		return 0, nil, 0, err
	}
	return midTableID, midKey, newRight.PageID(), nil
}

func (bt *BTree) createNewRoot(oldRootID uint32, tableID uint32, key types.Value, rightPageID uint32) error {
	newRoot, err := bt.disk.AllocatePage(page.TypeInternal)
	if err != nil {
		return err
	}
	// findChildPageID の規約では、セルの子ポインタはそのセルのキー未満を担当する。
	// したがって分割キー未満は旧ルート（左）、以上は新しい右ページに振り分ける。
	cell := encodeInternalCell(tableID, key, oldRootID)
	newRoot.AddCell(cell)
	newRoot.SetRightmostChild(rightPageID)

	if err := bt.disk.WritePage(newRoot); err != nil {
		return err
	}
	return bt.disk.SetRootPageID(newRoot.PageID())
}

func (bt *BTree) findLeftmostLeaf(pageID uint32) uint32 {
	for {
		p, err := bt.disk.ReadPage(pageID)
		if err != nil || p.Type() == page.TypeLeaf {
			return pageID
		}
		if p.CellCount() == 0 {
			return pageID
		}
		cell := p.CellAt(0)
		_, _, childID := decodeInternalCell(cell)
		pageID = childID
	}
}

// --- ページユーティリティ ---

func initLeafLinks(p *page.Page) {
	p.SetRightmostChild(page.NoPageID)
}

func nextLeafID(p *page.Page) uint32 {
	return p.RightmostChild()
}

func setNextLeafID(p *page.Page, id uint32) {
	p.SetRightmostChild(id)
}

func setPrevLeafID(p *page.Page, id uint32) {
	_ = id
}

func resetPage(p *page.Page) {
	b := p.Bytes()
	for i := page.HeaderSize; i < page.PageSize; i++ {
		b[i] = 0
	}
	binary.BigEndian.PutUint16(b[13:15], 0)
	binary.BigEndian.PutUint16(b[15:17], uint16(page.PageSize))
}
