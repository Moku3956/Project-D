package page

import (
	"bytes"
	"testing"
)

// ---- 正常系 ----

// NewPage が Type・PageID・CellCount・CellContentOffset を正しく初期化するか確認する。
func TestNewPage(t *testing.T) {
	p := NewPage(TypeLeaf, 5)

	if p.Type() != TypeLeaf {
		t.Errorf("expected TypeLeaf, got %d", p.Type())
	}
	if p.PageID() != 5 {
		t.Errorf("expected pageID=5, got %d", p.PageID())
	}
	if p.CellCount() != 0 {
		t.Errorf("expected cellCount=0, got %d", p.CellCount())
	}
	if p.CellContentOffset() != PageSize {
		t.Errorf("expected cellContentOffset=%d, got %d", PageSize, p.CellContentOffset())
	}
}

// Bytes でバイト列に変換し、FromBytes で復元してもヘッダフィールドが一致するか確認する。
func TestFromBytes(t *testing.T) {
	p := NewPage(TypeInternal, 3)
	p.SetLSN(42)

	p2 := FromBytes(p.Bytes())

	if p2.Type() != TypeInternal {
		t.Errorf("expected TypeInternal, got %d", p2.Type())
	}
	if p2.PageID() != 3 {
		t.Errorf("expected pageID=3, got %d", p2.PageID())
	}
	if p2.LSN() != 42 {
		t.Errorf("expected LSN=42, got %d", p2.LSN())
	}
}

// AddCell が成功し CellCount が増えることを確認する。
func TestAddCell(t *testing.T) {
	p := NewPage(TypeLeaf, 0)

	cell := []byte{1, 2, 3, 4}
	if !p.AddCell(cell) {
		t.Fatal("expected AddCell to succeed")
	}

	if p.CellCount() != 1 {
		t.Errorf("expected cellCount=1, got %d", p.CellCount())
	}
}

// CellAt が挿入順に正しいバイト列を返すことを確認する。
func TestCellAt(t *testing.T) {
	p := NewPage(TypeLeaf, 0)

	cell0 := []byte{1, 2, 3}
	cell1 := []byte{4, 5, 6, 7}
	p.AddCell(cell0)
	p.AddCell(cell1)

	got0 := p.CellAt(0)
	if string(got0) != string(cell0) {
		t.Errorf("CellAt(0): expected %v, got %v", cell0, got0)
	}

	got1 := p.CellAt(1)
	if string(got1) != string(cell1) {
		t.Errorf("CellAt(1): expected %v, got %v", cell1, got1)
	}
}

// DeleteCell がスロットを詰め、CellCount を減らすことを確認する。
// 論理削除のため物理データは残り、削除後の CellAt は先頭バイトのみ検証する。
func TestDeleteCell(t *testing.T) {
	p := NewPage(TypeLeaf, 0)

	p.AddCell([]byte{1, 2, 3})
	p.AddCell([]byte{4, 5, 6})
	p.AddCell([]byte{7, 8, 9})

	p.DeleteCell(1)

	if p.CellCount() != 2 {
		t.Errorf("expected cellCount=2, got %d", p.CellCount())
	}

	got0 := p.CellAt(0)
	if !bytes.Equal(got0, []byte{1, 2, 3}) {
		t.Errorf("CellAt(0) after delete: expected [1 2 3], got %v", got0)
	}

	got1 := p.CellAt(1)
	if len(got1) < 3 || !bytes.Equal(got1[:3], []byte{7, 8, 9}) {
		t.Errorf("CellAt(1) after delete: expected prefix [7 8 9], got %v", got1)
	}
}

// FreeSpace がセル+スロット分だけ正確に減ることを確認する。
func TestFreeSpace(t *testing.T) {
	p := NewPage(TypeLeaf, 0)

	initial := p.FreeSpace()
	expected := PageSize - HeaderSize
	if initial != expected {
		t.Errorf("expected initial FreeSpace=%d, got %d", expected, initial)
	}

	cell := make([]byte, 10)
	p.AddCell(cell)

	after := p.FreeSpace()
	if after != initial-12 {
		t.Errorf("expected FreeSpace=%d after AddCell, got %d", initial-12, after)
	}
}

// RightmostChild の読み書きが正しく機能することを確認する。
func TestRightmostChild(t *testing.T) {
	p := NewPage(TypeInternal, 0)
	p.SetRightmostChild(99)

	if p.RightmostChild() != 99 {
		t.Errorf("expected RightmostChild=99, got %d", p.RightmostChild())
	}
}

// ---- 異常系 ----

// PageSize と異なる長さのバイト列を FromBytes に渡すと panic することを確認する。
func TestFromBytesInvalidLength(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid length")
		}
	}()
	FromBytes(make([]byte, 10))
}

// セルが0件のページで DeleteCell を呼ぶと panic することを確認する。
func TestDeleteCellEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty page")
		}
	}()
	p := NewPage(TypeLeaf, 0)
	p.DeleteCell(0)
}

// CellCount 以上のインデックスで DeleteCell を呼ぶと panic することを確認する。
func TestDeleteCellOutOfRange(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for out-of-range index")
		}
	}()
	p := NewPage(TypeLeaf, 0)
	p.AddCell([]byte{1, 2, 3})
	p.DeleteCell(1)
}

// 空スライスを渡すと AddCell が false を返すことを確認する。
func TestAddCellEmpty(t *testing.T) {
	p := NewPage(TypeLeaf, 0)

	if p.AddCell([]byte{}) {
		t.Error("expected AddCell to reject empty cell")
	}
	if p.CellCount() != 0 {
		t.Errorf("expected cellCount=0, got %d", p.CellCount())
	}
}

// ページが満杯になると AddCell が false を返し、FreeSpace が負にならないことを確認する。
func TestAddCellFull(t *testing.T) {
	p := NewPage(TypeLeaf, 0)

	cell := make([]byte, 100)
	for p.AddCell(cell) {
	}

	if p.FreeSpace() < 0 {
		t.Error("FreeSpace should not be negative")
	}
}
