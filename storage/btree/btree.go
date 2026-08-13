package btree

import (
	"encoding/binary"
	"fmt"

	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/types"
)

// BTree はB+Treeの操作を提供する。
type BTree struct {
	disk   *page.DiskManager
	schema *types.Schema
}

func New(disk *page.DiskManager, schema *types.Schema) (*BTree, error) {
	bt := &BTree{disk: disk, schema: schema}

	// ルートが存在しない場合は空の葉ノードをルートとして作成
	if disk.RootPageID() == page.NoPageID {
		// nextPageIDが0のときルートも未作成
		info, err := disk.AllocatePage(page.TypeLeaf)
		if err != nil {
			return nil, err
		}
		// 葉ノードのprev/nextを0で初期化
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

// Search はキーに対応するRowを返す。見つからない場合はnil。
func (bt *BTree) Search(key types.Value) (*types.Row, error) {
	leafPage, err := bt.findLeaf(key)
	if err != nil {
		return nil, err
	}
	idx, found := bt.searchInLeaf(leafPage, key)
	if !found {
		return nil, nil
	}
	cell := leafPage.CellAt(idx)
	_, row := decodeLeafCell(cell, bt.schema)
	return &row, nil
}

// Insert はキーとRowを挿入する。
func (bt *BTree) Insert(key types.Value, row types.Row) error {
	rootID := bt.disk.RootPageID()
	newKey, newPageID, err := bt.insertRecursive(rootID, key, row)
	if err != nil {
		return err
	}
	// ルートが分割された場合
	if newPageID != 0 {
		if err := bt.createNewRoot(rootID, newKey, newPageID); err != nil {
			return err
		}
	}
	return nil
}

// Delete はキーに対応するレコードを削除する。
func (bt *BTree) Delete(key types.Value) error {
	leafPage, err := bt.findLeaf(key)
	if err != nil {
		return err
	}
	idx, found := bt.searchInLeaf(leafPage, key)
	if !found {
		return fmt.Errorf("key not found")
	}
	leafPage.DeleteCell(idx)
	return bt.disk.WritePage(leafPage)
}

// Scan は全レコードを葉ノードのリンクリストを辿って返す。
func (bt *BTree) Scan() ([]types.Row, error) {
	rootID := bt.disk.RootPageID()

	// リンクリストで全葉を辿る（nextはRightmostChildを流用）
	var rows []types.Row
	leafID := bt.findLeftmostLeaf(rootID)
	for leafID != page.NoPageID {
		p, err := bt.disk.ReadPage(leafID)
		if err != nil {
			return nil, err
		}
		n := int(p.CellCount())
		for i := 0; i < n; i++ {
			cell := p.CellAt(i)
			_, row := decodeLeafCell(cell, bt.schema)
			rows = append(rows, row)
		}
		leafID = nextLeafID(p)
	}
	return rows, nil
}

// --- 内部実装 ---

func (bt *BTree) pkKind() types.DataTypeKind {
	return bt.schema.Columns[bt.schema.PrimaryKeyIndex()].Type.Kind
}

func (bt *BTree) findLeaf(key types.Value) (*page.Page, error) {
	pageID := bt.disk.RootPageID()
	for {
		p, err := bt.disk.ReadPage(pageID)
		if err != nil {
			return nil, err
		}
		if p.Type() == page.TypeLeaf {
			return p, nil
		}
		pageID = bt.findChildPageID(p, key)
	}
}

func (bt *BTree) findChildPageID(p *page.Page, key types.Value) uint32 {
	n := int(p.CellCount())
	for i := 0; i < n; i++ {
		cell := p.CellAt(i)
		k, childID := decodeInternalCell(cell, bt.pkKind())
		if compareKeys(key, k) < 0 {
			// keyはchildIDの左の子にある
			// 左の子はi==0のときは別途管理が必要だが、
			// ここではchildIDをそのまま使う（右の子）
			// 正確にはi==0の場合は左子ポインタが必要
			_ = childID
			if i == 0 {
				// 最左子: スロット配列の前に左ポインタは今回は右端で代用
				// TODO: 正確な実装
				return childID
			}
			prevCell := p.CellAt(i - 1)
			_, prevChild := decodeInternalCell(prevCell, bt.pkKind())
			return prevChild
		}
	}
	return p.RightmostChild()
}

func (bt *BTree) searchInLeaf(p *page.Page, key types.Value) (int, bool) {
	n := int(p.CellCount())
	for i := 0; i < n; i++ {
		cell := p.CellAt(i)
		k, _ := decodeLeafCell(cell, bt.schema)
		cmp := compareKeys(k, key)
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
// 分割が発生した場合は(分割キー, 新しい右ページID)を返す。
func (bt *BTree) insertRecursive(pageID uint32, key types.Value, row types.Row) (types.Value, uint32, error) {
	p, err := bt.disk.ReadPage(pageID)
	if err != nil {
		return nil, 0, err
	}

	if p.Type() == page.TypeLeaf {
		return bt.insertIntoLeaf(p, key, row)
	}
	return bt.insertIntoInternal(p, key, row)
}

func (bt *BTree) insertIntoLeaf(p *page.Page, key types.Value, row types.Row) (types.Value, uint32, error) {
	cell := encodeLeafCell(key, row, bt.schema)
	if p.AddCell(cell) {
		return nil, 0, bt.disk.WritePage(p)
	}
	return bt.splitLeaf(p, key, row)
}

func (bt *BTree) insertIntoInternal(p *page.Page, key types.Value, row types.Row) (types.Value, uint32, error) {
	childID := bt.findChildPageID(p, key)
	upKey, newChildID, err := bt.insertRecursive(childID, key, row)
	if err != nil {
		return nil, 0, err
	}
	if newChildID == 0 {
		return nil, 0, nil
	}
	// 子が分割された → このノードにupKeyを挿入
	cell := encodeInternalCell(upKey, newChildID)
	if p.AddCell(cell) {
		return nil, 0, bt.disk.WritePage(p)
	}
	return bt.splitInternal(p, upKey, newChildID)
}

func (bt *BTree) splitLeaf(p *page.Page, key types.Value, row types.Row) (types.Value, uint32, error) {
	// 新しい右ページを作成
	rightPage, err := bt.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		return nil, 0, err
	}
	initLeafLinks(rightPage)

	// 既存セルを全部取り出す
	n := int(p.CellCount())
	cells := make([][]byte, n)
	for i := 0; i < n; i++ {
		c := p.CellAt(i)
		cp := make([]byte, len(c))
		copy(cp, c)
		cells[i] = cp
	}

	// 新セルを追加してソート
	newCell := encodeLeafCell(key, row, bt.schema)
	cells = append(cells, newCell)
	sortCells(cells, func(c []byte) types.Value {
		k, _ := decodeLeafCell(c, bt.schema)
		return k
	})

	// 左右に分割
	mid := len(cells) / 2
	resetPage(p)
	for _, c := range cells[:mid] {
		p.AddCell(c)
	}
	for _, c := range cells[mid:] {
		rightPage.AddCell(c)
	}

	// リンクリストを更新
	setNextLeafID(p, rightPage.PageID())
	setPrevLeafID(rightPage, p.PageID())

	// 分割キーは右ページの最小キー
	splitKey, _ := decodeLeafCell(cells[mid], bt.schema)

	if err := bt.disk.WritePage(p); err != nil {
		return nil, 0, err
	}
	if err := bt.disk.WritePage(rightPage); err != nil {
		return nil, 0, err
	}
	return splitKey, rightPage.PageID(), nil
}

func (bt *BTree) splitInternal(p *page.Page, key types.Value, rightChildID uint32) (types.Value, uint32, error) {
	newRight, err := bt.disk.AllocatePage(page.TypeInternal)
	if err != nil {
		return nil, 0, err
	}

	n := int(p.CellCount())
	cells := make([][]byte, n)
	for i := 0; i < n; i++ {
		c := p.CellAt(i)
		cp := make([]byte, len(c))
		copy(cp, c)
		cells[i] = cp
	}
	newCell := encodeInternalCell(key, rightChildID)
	cells = append(cells, newCell)
	sortCells(cells, func(c []byte) types.Value {
		k, _ := decodeInternalCell(c, bt.pkKind())
		return k
	})

	mid := len(cells) / 2
	midKey, _ := decodeInternalCell(cells[mid], bt.pkKind())

	resetPage(p)
	for _, c := range cells[:mid] {
		p.AddCell(c)
	}
	// 右端の子ポインタを設定
	_, midChild := decodeInternalCell(cells[mid], bt.pkKind())
	p.SetRightmostChild(midChild)

	for _, c := range cells[mid+1:] {
		newRight.AddCell(c)
	}
	_, lastChild := decodeInternalCell(cells[len(cells)-1], bt.pkKind())
	newRight.SetRightmostChild(lastChild)

	if err := bt.disk.WritePage(p); err != nil {
		return nil, 0, err
	}
	if err := bt.disk.WritePage(newRight); err != nil {
		return nil, 0, err
	}
	return midKey, newRight.PageID(), nil
}

func (bt *BTree) createNewRoot(oldRootID uint32, key types.Value, rightPageID uint32) error {
	newRoot, err := bt.disk.AllocatePage(page.TypeInternal)
	if err != nil {
		return err
	}
	cell := encodeInternalCell(key, rightPageID)
	newRoot.AddCell(cell)
	newRoot.SetRightmostChild(rightPageID)

	// 左子ポインタとして旧ルートを保持
	// 内部ノードのcell[0]の左子 = oldRootID
	// 今の実装ではcell[0]のchildIDが右子なので、leftmostは別途必要
	// 簡略化: oldRootIDをfreelistHeadに一時保存（TODO: 正式な左子ポインタ）
	_ = oldRootID

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
		_, childID := decodeInternalCell(cell, bt.pkKind())
		pageID = childID
	}
}

// --- ページユーティリティ ---

// 葉ノードのnextPageIDはRightmostChildフィールドを流用する（葉では子ポインタ不要）。

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
	// セルカウントとセルコンテンツオフセットをリセット
	binary.BigEndian.PutUint16(b[13:15], 0)
	binary.BigEndian.PutUint16(b[15:17], uint16(page.PageSize))
}

func sortCells(cells [][]byte, keyFn func([]byte) types.Value) {
	// 単純バブルソート
	for i := 0; i < len(cells); i++ {
		for j := 0; j < len(cells)-1-i; j++ {
			if compareKeys(keyFn(cells[j]), keyFn(cells[j+1])) > 0 {
				cells[j], cells[j+1] = cells[j+1], cells[j]
			}
		}
	}
}
