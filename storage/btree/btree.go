package btree

import (
	"encoding/binary"
	"fmt"

	"github.com/Moku3956/Project-D/storage/buffer"
	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
	"github.com/Moku3956/Project-D/types"
)

// BTree はB+Treeの操作を提供する。全テーブルで単一インスタンスを共有する。
type BTree struct {
	disk *page.DiskManager  // ページ確保・ルートページID管理用(バッファプールが持たない操作)
	bp   *buffer.BufferPool // 既存ページの読み書きはすべてここを経由する(No-Steal)
	wm   *wal.WALManager    // ページ変更のRedoログ記録用
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

// Search はtableIDとキーに対応するRowを返す。見つからない場合はnil。
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

// Insert はtableIDとキーとRowを挿入する。txnIDはWALのRedoログに記録される。
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

// Delete はtableIDとキーに対応するレコードを削除する。
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

// Scan はtableIDに属する全レコードを葉ノードのリンクリストを辿って返す。
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

// --- 内部実装 ---

// finishPage はページの変更をWALにRedoログとして記録し、バッファプール上でdirtyにする。
// ディスクへの実書き込みはコミット時まで行わない(No-Steal)。WAL追記に失敗した場合は
// 変更を破棄扱いでアンピンする。
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

// releasePage は変更を加えずにページをアンピンする。
func (bt *BTree) releasePage(p *page.Page) {
	bt.bp.UnpinPage(p.PageID(), false, 0)
}

// findLeaf はtableID・keyが入っているはずの葉ページのIDを返す。
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

// serchInLeafは探しているtable, keyに対応するセルを指定された(絞り済み)ページの中から探す。
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

// insertIntoLeaf はpの葉ページに挿入する。呼び出し元がFetch済みのpのアンピンはこの関数(またはsplitLeaf)が担う。
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

// insertIntoInternal はpの内部ノードに挿入する。呼び出し元がFetch済みのpのアンピンはこの関数(またはsplitInternal)が担う。
func (bt *BTree) insertIntoInternal(p *page.Page, tableID uint32, key types.Value, row types.Row, schema *types.Schema, txnID uint64) (uint32, types.Value, uint32, error) {
	childID := bt.findChildPageID(p, tableID, key)
	upTableID, upKey, newChildID, err := bt.insertRecursive(childID, tableID, key, row, schema, txnID)
	if err != nil {
		bt.releasePage(p)
		return 0, nil, 0, err
	}
	if newChildID == 0 {
		bt.releasePage(p)
		return 0, nil, 0, nil
	}

	// childID(分割で縮んだ古い子)が占めていたポインタをnewChildIDに差し替える。
	// キーは変わらずchildだけ変わるためセル長は同じで、その場で上書きできる。
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

	// childID(古い子。範囲が縮んだ)をupKey未満の担当として新しいセルで追加する。
	cell := encodeInternalCell(upTableID, upKey, childID)
	if p.InsertCellAt(bt.findInsertPos(p, upTableID, upKey), cell) {
		if err := bt.finishPage(p, txnID, wal.OpInsert); err != nil {
			return 0, nil, 0, err
		}
		return 0, nil, 0, nil
	}
	return bt.splitInternal(p, upTableID, upKey, childID, txnID)
}

// splitLeaf はpを引き取り、分割してrightPageを新設する。p・rightPage両方のアンピンをここで行う。
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

	// 分割前にpが持っていた次のポインタを保持
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
	// prev方向は未実装（呼び出し元がないため一旦コメントアウト）。ページヘッダに専用フィールドがなく、
	// 実装するにはヘッダ拡張が必要。storage/btree/docs/spec.md参照。
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

// splitInternal はpを引き取り、分割してnewRightを新設する。p・newRight両方のアンピンをここで行う。
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

	// 分割前にpが持っていたRightmostChildを退避する。これはどのキーとも組まないため、
	// セル配列(cells)には含まれず、分割後は右ページがそのまま引き継ぐ。
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
	// findChildPageID の規約では、セルの子ポインタはそのセルのキー未満を担当する。
	// したがって分割キー未満は旧ルート（左）、以上は新しい右ページに振り分ける。
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

// --- ページユーティリティ ---

func initLeafLinks(p *page.Page) {
	p.SetRightmostChild(page.NoPageID)
}

// nextLeafIDは次のリーフノードのポインタを返す。
func nextLeafID(p *page.Page) uint32 {
	return p.RightmostChild()
}

func setNextLeafID(p *page.Page, id uint32) {
	p.SetRightmostChild(id)
}

// prev方向は未実装のため一旦コメントアウト。呼び出し元(splitLeaf)もコメントアウト済み。
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
