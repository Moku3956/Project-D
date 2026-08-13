package page

import "encoding/binary"

const (
	PageSize       = 4096
	HeaderSize     = 24
	FileHeaderSize = 4096

	TypeInternal uint8 = 0x01
	TypeLeaf     uint8 = 0x02
)

// Page はディスク上の4KBブロック。
type Page struct {
	data [PageSize]byte
}

func NewPage(pageType uint8, pageID uint32) *Page {
	p := &Page{}
	p.setType(pageType)
	p.setPageID(pageID)
	p.setCellContentOffset(PageSize)
	return p
}

// バイト列をページに変換
func FromBytes(b []byte) *Page {
	if len(b) != PageSize {
		panic("FromBytes: invalid length")
	}
	p := &Page{}
	copy(p.data[:], b)
	return p
}

// ページデータをバイト列として返す
func (p *Page) Bytes() []byte { return p.data[:] }

// ヘッダフィールドのgetter/setter

// ページの種別(内部ノードかリーフノード)を取得、設定
func (p *Page) Type() uint8     { return p.data[0] }
func (p *Page) setType(t uint8) { p.data[0] = t }

// ページ番号を取得。設定
func (p *Page) PageID() uint32      { return binary.BigEndian.Uint32(p.data[1:5]) }
func (p *Page) setPageID(id uint32) { binary.BigEndian.PutUint32(p.data[1:5], id) }

// シーケンス番号を取得、設定
func (p *Page) LSN() uint64       { return binary.BigEndian.Uint64(p.data[5:13]) }
func (p *Page) SetLSN(lsn uint64) { binary.BigEndian.PutUint64(p.data[5:13], lsn) }

// セル数(KV数)を取得、設定
func (p *Page) CellCount() uint16     { return binary.BigEndian.Uint16(p.data[13:15]) }
func (p *Page) setCellCount(n uint16) { binary.BigEndian.PutUint16(p.data[13:15], n) }

// セルの開始オフセットを取得、設定
func (p *Page) CellContentOffset() uint16       { return binary.BigEndian.Uint16(p.data[15:17]) }
func (p *Page) setCellContentOffset(off uint16) { binary.BigEndian.PutUint16(p.data[15:17], off) }

// フリーリストの開始オフセットを取得、設定
func (p *Page) FreelistHead() uint16              { return binary.BigEndian.Uint16(p.data[17:19]) }
// func (p *Page) setFreelistHead(off uint16)        { binary.BigEndian.PutUint16(p.data[17:19], off) }

// 断片化バイト数を取得、設定
func (p *Page) FragmentedBytes() uint8    { return p.data[19] }
// func (p *Page) setFragmentedBytes(n uint8) { p.data[19] = n }

// 右端の子ポインタを取得、設定
func (p *Page) RightmostChild() uint32      { return binary.BigEndian.Uint32(p.data[20:24]) }
func (p *Page) SetRightmostChild(id uint32) { binary.BigEndian.PutUint32(p.data[20:24], id) }

// スロット配列: HeaderSize + i*2 の位置に各セルのオフセットを格納
// セル(KV)のオフセットを取得、設定
func (p *Page) slotOffset(i int) int { return HeaderSize + i*2 }
func (p *Page) GetSlot(i int) uint16 {
	off := p.slotOffset(i)
	return binary.BigEndian.Uint16(p.data[off : off+2])
}
func (p *Page) setSlot(i int, cellOff uint16) {
	off := p.slotOffset(i)
	binary.BigEndian.PutUint16(p.data[off:off+2], cellOff)
}

// FreeSpace はスロット配列とセルデータの間の空き領域サイズを返す。
func (p *Page) FreeSpace() int {
	slotEnd := HeaderSize + int(p.CellCount())*2
	return int(p.CellContentOffset()) - slotEnd
}

// AddCell はセルデータをページ末尾側に書き込み、スロットを追加する。
func (p *Page) AddCell(cell []byte) bool {
	// +2はオフセット分
	if len(cell) == 0 || p.FreeSpace() < len(cell)+2 {
		return false
	}
	newOff := p.CellContentOffset() - uint16(len(cell))
	copy(p.data[newOff:], cell)
	p.setCellContentOffset(newOff)
	n := p.CellCount()
	p.setSlot(int(n), newOff)
	p.setCellCount(n + 1)
	return true
}

// i番目のセルデータを返す(セルフォーマットを見ないと若干複雑かも？)
func (p *Page) CellAt(i int) []byte {
	off := p.GetSlot(i)
	// セルサイズはスロットの次のオフセットとの差か末尾から計算
	// 終端: 次スロットが指す位置またはPageSize
	var end uint16
	if i == 0 {
		end = PageSize
	} else {
		end = p.GetSlot(i - 1)
	}
	return p.data[off:end]
}

// DeleteCell はi番目のセルを論理削除する（スロットを詰める）。
func (p *Page) DeleteCell(i int) {
	n := int(p.CellCount())
	if n == 0 || i < 0 || i >= n {
		panic("DeleteCell: index out of range")
	}
	for j := i; j < n-1; j++ {
		p.setSlot(j, p.GetSlot(j+1))
	}
	p.setCellCount(uint16(n - 1))
}
