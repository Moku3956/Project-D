package btree

import (
	"encoding/binary"
	"fmt"

	"github.com/Moku3956/Project-D/storage/buffer"
	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
	"github.com/Moku3956/Project-D/types"
)

// BTree provides B+Tree operations. A single instance is shared across all tables.
type BTree struct {
	disk *page.DiskManager  // for page allocation and root page ID management (operations the buffer pool doesn't provide)
	bp   *buffer.BufferPool // all reads/writes of existing pages go through here (No-Steal)
	wm   *wal.WALManager    // for recording redo logs of page changes
}

func NewBTree(disk *page.DiskManager, bp *buffer.BufferPool, wm *wal.WALManager) (*BTree, error) {
	bt := &BTree{disk: disk, bp: bp, wm: wm}

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

// Search returns the Row for tableID and key. Returns nil if not found.
func (bt *BTree) Search(tableID uint32, key types.Value, schema *types.Schema) (*types.Row, error) {
	leafID, err := bt.findLeaf(tableID, key)
	if err != nil {
		return nil, err
	}
	p, err := bt.bp.FetchPage(leafID)
	if err != nil {
		return nil, err
	}
	defer bt.releasePage(p)

	idx, found := bt.searchInLeaf(p, tableID, key)
	if !found {
		return nil, nil
	}
	cell := p.CellAt(idx)
	_, _, row := decodeLeafCell(cell, schema)
	return &row, nil
}

// Insert inserts tableID, key, and row. txnID is recorded in the WAL redo log.
func (bt *BTree) Insert(tableID uint32, key types.Value, row types.Row, schema *types.Schema, txnID uint64) error {
	rootID := bt.disk.RootPageID()
	upTableID, upKey, newPageID, err := bt.insertRecursive(rootID, tableID, key, row, schema, txnID)
	if err != nil {
		return err
	}
	if newPageID != 0 {
		return bt.createNewRoot(rootID, upTableID, upKey, newPageID, txnID)
	}
	return nil
}

// Update はtableIDとキーに対応する行をnewRowで置き換える。PRIMARY KEYは変わらない
// ため対象の葉ページは元の行と同じで、よくあるケース(古いセルを消した後にnewRowが
// 同じページに収まる)は1回のページ変更で済み、OpUpdateレコード1件だけがWALに残る。
// (単純にDelete+Insertを呼ぶと、同じ内容でも常に2件のレコードになってしまう。)
func (bt *BTree) Update(tableID uint32, key types.Value, newRow types.Row, schema *types.Schema, txnID uint64) error {
	rootID := bt.disk.RootPageID()
	upTableID, upKey, newPageID, err := bt.updateRecursive(rootID, tableID, key, newRow, schema, txnID)
	if err != nil {
		return err
	}
	if newPageID != 0 {
		return bt.createNewRoot(rootID, upTableID, upKey, newPageID, txnID)
	}
	return nil
}

// Delete removes the record for tableID and key.
func (bt *BTree) Delete(tableID uint32, key types.Value, txnID uint64) error {
	leafID, err := bt.findLeaf(tableID, key)
	if err != nil {
		return err
	}
	p, err := bt.bp.FetchPage(leafID)
	if err != nil {
		return err
	}
	idx, found := bt.searchInLeaf(p, tableID, key)
	if !found {
		bt.releasePage(p)
		return fmt.Errorf("key not found")
	}
	p.DeleteCell(idx)
	return bt.finishPage(p, txnID, wal.OpDelete)
}

// Scan returns all records belonging to tableID by walking the leaf nodes' linked list.
func (bt *BTree) Scan(tableID uint32, schema *types.Schema) ([]types.Row, error) {
	var rows []types.Row
	leafID, err := bt.findLeftmostLeaf(bt.disk.RootPageID())
	if err != nil {
		return nil, err
	}
	for leafID != page.NoPageID {
		p, err := bt.bp.FetchPage(leafID)
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
		next := nextLeafID(p)
		bt.releasePage(p)
		if done {
			break
		}
		leafID = next
	}
	return rows, nil
}

// --- Internal implementation ---

// finishPage saves the log to wm.buf and marks the page dirty in bp.
func (bt *BTree) finishPage(p *page.Page, txnID uint64, op wal.Operation) error {
	lsn, err := bt.wm.Append(&wal.LogRecord{
		TxnID:    txnID,
		PageID:   p.PageID(),
		Op:       op,
		RedoData: append([]byte(nil), p.Bytes()...),
	})
	if err != nil {
		bt.bp.UnpinPage(p.PageID(), false, 0)
		return err
	}
	p.SetLSN(lsn)
	bt.bp.UnpinPage(p.PageID(), true, txnID)
	return nil
}

// releasePage unpins the page without marking it as changed.
func (bt *BTree) releasePage(p *page.Page) {
	bt.bp.UnpinPage(p.PageID(), false, 0)
}

// findLeaf returns the ID of the leaf page that should contain tableID and key.
func (bt *BTree) findLeaf(tableID uint32, key types.Value) (uint32, error) {
	pageID := bt.disk.RootPageID()
	for {
		p, err := bt.bp.FetchPage(pageID)
		if err != nil {
			return 0, err
		}
		if p.Type() == page.TypeLeaf {
			bt.releasePage(p)
			return pageID, nil
		}
		next := bt.findChildPageID(p, tableID, key)
		bt.releasePage(p)
		pageID = next
	}
}

func (bt *BTree) findChildPageID(p *page.Page, tableID uint32, key types.Value) uint32 {
	n := int(p.CellCount())
	for i := 0; i < n; i++ {
		cell := p.CellAt(i)
		kid, k, childID := decodeInternalCell(cell)
		if compareCompositeKeys(tableID, key, kid, k) < 0 {
			return childID
		}
	}
	return p.RightmostChild()
}

// searchInLeaf looks for the cell matching the given table and key within the
// specified (already narrowed-down) page.
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

// insertRecursive inserts key/row into the subtree rooted at pageID.
// If a split occurs, it returns (split tableID, split key, new right page ID).
func (bt *BTree) insertRecursive(pageID uint32, tableID uint32, key types.Value, row types.Row, schema *types.Schema, txnID uint64) (uint32, types.Value, uint32, error) {
	p, err := bt.bp.FetchPage(pageID)
	if err != nil {
		return 0, nil, 0, err
	}
	if p.Type() == page.TypeLeaf {
		return bt.insertIntoLeaf(p, tableID, key, row, schema, txnID)
	}
	return bt.insertIntoInternal(p, tableID, key, row, schema, txnID)
}

// findInsertPos returns the insertion position that preserves composite key order.
// Works for both leaf and internal cells, since both start with a composite key.
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

// insertIntoLeaf inserts into leaf page p. Unpinning the already-fetched p is
// the responsibility of this function (or splitLeaf).
func (bt *BTree) insertIntoLeaf(p *page.Page, tableID uint32, key types.Value, row types.Row, schema *types.Schema, txnID uint64) (uint32, types.Value, uint32, error) {
	cell := encodeLeafCell(tableID, key, row, schema)
	if p.InsertCellAt(bt.findInsertPos(p, tableID, key), cell) {
		if err := bt.finishPage(p, txnID, wal.OpInsert); err != nil {
			return 0, nil, 0, err
		}
		return 0, nil, 0, nil
	}
	return bt.splitLeaf(p, tableID, key, row, schema, txnID)
}

// insertIntoInternal inserts into internal node p. Unpinning the already-fetched
// p is the responsibility of this function (or splitInternal).
func (bt *BTree) insertIntoInternal(p *page.Page, tableID uint32, key types.Value, row types.Row, schema *types.Schema, txnID uint64) (uint32, types.Value, uint32, error) {
	childID := bt.findChildPageID(p, tableID, key)
	upTableID, upKey, newChildID, err := bt.insertRecursive(childID, tableID, key, row, schema, txnID)
	if err != nil {
		bt.releasePage(p)
		return 0, nil, 0, err
	}
	return bt.patchInternalAfterChildSplit(p, childID, upTableID, upKey, newChildID, txnID)
}

// patchInternalAfterChildSplit は、子ページ(childID)を処理した結果 newChildID
// (分割が起きていれば新しい右ページのID、起きていなければ0)を受け取り、pの
// 子ポインタ・セルを更新する。insertIntoInternalとupdateIntoInternalの共通部分
// (「子を再帰処理した後にpをどう直すか」はInsertでもUpdateでも全く同じ)。
func (bt *BTree) patchInternalAfterChildSplit(p *page.Page, childID uint32, upTableID uint32, upKey types.Value, newChildID uint32, txnID uint64) (uint32, types.Value, uint32, error) {
	if newChildID == 0 {
		bt.releasePage(p)
		return 0, nil, 0, nil
	}

	// Replace the pointer that childID (the old child, now shrunk by the split)
	// occupied with newChildID. The key stays the same and only the child
	// changes, so the cell length is unchanged and can be overwritten in place.
	replaced := false
	n := int(p.CellCount())
	for i := 0; i < n; i++ {
		cellBytes := p.CellAt(i)
		kid, k, cid := decodeInternalCell(cellBytes)
		if cid == childID {
			copy(cellBytes, encodeInternalCell(kid, k, newChildID))
			replaced = true
			break
		}
	}
	if !replaced {
		p.SetRightmostChild(newChildID)
	}

	// Add a new cell so that childID (the old child, whose range has shrunk) is
	// responsible for keys less than upKey.
	cell := encodeInternalCell(upTableID, upKey, childID)
	if p.InsertCellAt(bt.findInsertPos(p, upTableID, upKey), cell) {
		if err := bt.finishPage(p, txnID, wal.OpInsert); err != nil {
			return 0, nil, 0, err
		}
		return 0, nil, 0, nil
	}
	return bt.splitInternal(p, upTableID, upKey, childID, txnID)
}

// updateRecursive はpageIDのサブツリーの中からtableID・keyの行を探してnewRowに
// 置き換える。insertRecursiveと同じ形の再帰構造で、葉に着くまで内部ノードを辿る。
func (bt *BTree) updateRecursive(pageID uint32, tableID uint32, key types.Value, newRow types.Row, schema *types.Schema, txnID uint64) (uint32, types.Value, uint32, error) {
	p, err := bt.bp.FetchPage(pageID)
	if err != nil {
		return 0, nil, 0, err
	}
	if p.Type() == page.TypeLeaf {
		return bt.updateLeaf(p, tableID, key, newRow, schema, txnID)
	}
	return bt.updateIntoInternal(p, tableID, key, newRow, schema, txnID)
}

// updateIntoInternal はpの内部ノードを辿ってchildを再帰的に更新する。
// 子の再帰処理後にpを直す部分はinsertIntoInternalと共通(patchInternalAfterChildSplit)。
func (bt *BTree) updateIntoInternal(p *page.Page, tableID uint32, key types.Value, newRow types.Row, schema *types.Schema, txnID uint64) (uint32, types.Value, uint32, error) {
	childID := bt.findChildPageID(p, tableID, key)
	upTableID, upKey, newChildID, err := bt.updateRecursive(childID, tableID, key, newRow, schema, txnID)
	if err != nil {
		bt.releasePage(p)
		return 0, nil, 0, err
	}
	return bt.patchInternalAfterChildSplit(p, childID, upTableID, upKey, newChildID, txnID)
}

// updateLeaf はpの葉ページの中でtableID・keyに対応するセルをnewRowで置き換える。
// 古いセルを消してから新しいセルを同じページに挿し直せれば1回のページ変更(OpUpdate)
// で済む。newRowが育ってそれでも収まらない場合だけ、Insertと同じsplitLeafに委ねる
// (splitLeafはp上の残りのセル+新セルをまとめて再配置するので、事前にDeleteCell
// しておけば古い値が二重に残ることはない)。
func (bt *BTree) updateLeaf(p *page.Page, tableID uint32, key types.Value, newRow types.Row, schema *types.Schema, txnID uint64) (uint32, types.Value, uint32, error) {
	idx, found := bt.searchInLeaf(p, tableID, key)
	if !found {
		bt.releasePage(p)
		return 0, nil, 0, fmt.Errorf("key not found")
	}
	p.DeleteCell(idx)

	newCell := encodeLeafCell(tableID, key, newRow, schema)
	if p.InsertCellAt(bt.findInsertPos(p, tableID, key), newCell) {
		if err := bt.finishPage(p, txnID, wal.OpUpdate); err != nil {
			return 0, nil, 0, err
		}
		return 0, nil, 0, nil
	}
	return bt.splitLeaf(p, tableID, key, newRow, schema, txnID)
}

// splitLeaf takes ownership of p, splits it, and creates a new rightPage. Both
// p and rightPage are unpinned here.
func (bt *BTree) splitLeaf(p *page.Page, tableID uint32, key types.Value, row types.Row, schema *types.Schema, txnID uint64) (uint32, types.Value, uint32, error) {
	rightAlloc, err := bt.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		bt.releasePage(p)
		return 0, nil, 0, err
	}
	rightPage, err := bt.bp.FetchPage(rightAlloc.PageID())
	if err != nil {
		bt.releasePage(p)
		return 0, nil, 0, err
	}
	initLeafLinks(rightPage)

	// Keep the "next" pointer p had before the split.
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

	// Re-link as p → rightPage → (p's original next).
	setNextLeafID(rightPage, oldNext)
	setNextLeafID(p, rightPage.PageID())
	// The prev direction is not implemented (commented out for now, since there's
	// no caller). The page header has no dedicated field for it; implementing
	// this would require extending the header. See storage/btree/docs/spec.md.
	// setPrevLeafID(rightPage, p.PageID())

	splitTableID, splitKey, _ := decodeCompositeKey(cells[mid])

	if err := bt.finishPage(p, txnID, wal.OpInsert); err != nil {
		bt.releasePage(rightPage)
		return 0, nil, 0, err
	}
	if err := bt.finishPage(rightPage, txnID, wal.OpInsert); err != nil {
		return 0, nil, 0, err
	}
	return splitTableID, splitKey, rightPage.PageID(), nil
}

// splitInternal takes ownership of p, splits it, and creates a new newRight.
// Both p and newRight are unpinned here.
func (bt *BTree) splitInternal(p *page.Page, tableID uint32, key types.Value, rightChildID uint32, txnID uint64) (uint32, types.Value, uint32, error) {
	newRightAlloc, err := bt.disk.AllocatePage(page.TypeInternal)
	if err != nil {
		bt.releasePage(p)
		return 0, nil, 0, err
	}
	newRight, err := bt.bp.FetchPage(newRightAlloc.PageID())
	if err != nil {
		bt.releasePage(p)
		return 0, nil, 0, err
	}

	// Save the RightmostChild p had before the split. Since it isn't paired
	// with any key, it's not included in the cells array; after the split,
	// the right page simply inherits it.
	oldRightmost := p.RightmostChild()

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
	newRight.SetRightmostChild(oldRightmost)

	if err := bt.finishPage(p, txnID, wal.OpInsert); err != nil {
		bt.releasePage(newRight)
		return 0, nil, 0, err
	}
	if err := bt.finishPage(newRight, txnID, wal.OpInsert); err != nil {
		return 0, nil, 0, err
	}
	return midTableID, midKey, newRight.PageID(), nil
}

func (bt *BTree) createNewRoot(oldRootID uint32, tableID uint32, key types.Value, rightPageID uint32, txnID uint64) error {
	newRootAlloc, err := bt.disk.AllocatePage(page.TypeInternal)
	if err != nil {
		return err
	}
	newRoot, err := bt.bp.FetchPage(newRootAlloc.PageID())
	if err != nil {
		return err
	}
	// Per findChildPageID's convention, a cell's child pointer is responsible
	// for keys less than that cell's key. So keys below the split key go to
	// the old root (left), and keys at or above it go to the new right page.
	cell := encodeInternalCell(tableID, key, oldRootID)
	newRoot.AddCell(cell)
	newRoot.SetRightmostChild(rightPageID)

	if err := bt.finishPage(newRoot, txnID, wal.OpInsert); err != nil {
		return err
	}
	return bt.disk.SetRootPageID(newRoot.PageID())
}

func (bt *BTree) findLeftmostLeaf(pageID uint32) (uint32, error) {
	for {
		p, err := bt.bp.FetchPage(pageID)
		if err != nil {
			return 0, err
		}
		if p.Type() == page.TypeLeaf || p.CellCount() == 0 {
			bt.releasePage(p)
			return pageID, nil
		}
		cell := p.CellAt(0)
		_, _, childID := decodeInternalCell(cell)
		bt.releasePage(p)
		pageID = childID
	}
}

// --- Page utilities ---

func initLeafLinks(p *page.Page) {
	p.SetRightmostChild(page.NoPageID)
}

// nextLeafID returns the pointer to the next leaf node.
func nextLeafID(p *page.Page) uint32 {
	return p.RightmostChild()
}

func setNextLeafID(p *page.Page, id uint32) {
	p.SetRightmostChild(id)
}

// The prev direction is not implemented, so this is commented out for now.
// The caller (splitLeaf) is also commented out.
// func setPrevLeafID(p *page.Page, id uint32) {
// 	_ = id
// }

func resetPage(p *page.Page) {
	b := p.Bytes()
	for i := page.HeaderSize; i < page.PageSize; i++ {
		b[i] = 0
	}
	binary.BigEndian.PutUint16(b[13:15], 0)
	binary.BigEndian.PutUint16(b[15:17], uint16(page.PageSize))
}
