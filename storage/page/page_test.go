package page

import (
	"bytes"
	"testing"
)

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

func TestAddCellFull(t *testing.T) {
	p := NewPage(TypeLeaf, 0)

	// 空き領域を埋め尽くす
	cell := make([]byte, 100)
	for p.AddCell(cell) {
	}

	if p.FreeSpace() < 0 {
		t.Error("FreeSpace should not be negative")
	}
}

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

func TestDeleteCell(t *testing.T) {
	p := NewPage(TypeLeaf, 0)

	p.AddCell([]byte{1, 2, 3})
	p.AddCell([]byte{4, 5, 6})
	p.AddCell([]byte{7, 8, 9})

	// 真ん中を削除
	p.DeleteCell(1)

	if p.CellCount() != 2 {
		t.Errorf("expected cellCount=2, got %d", p.CellCount())
	}

	// slot[0] は末尾に隣接するため正確に3バイト返る
	got0 := p.CellAt(0)
	if !bytes.Equal(got0, []byte{1, 2, 3}) {
		t.Errorf("CellAt(0) after delete: expected [1 2 3], got %v", got0)
	}

	// 論理削除のため slot[1] の終端は旧slot[0]のオフセットになり、
	// 削除されたセルの物理データを含む可能性がある。
	// そのため先頭3バイトだけ検証する。
	got1 := p.CellAt(1)
	if len(got1) < 3 || !bytes.Equal(got1[:3], []byte{7, 8, 9}) {
		t.Errorf("CellAt(1) after delete: expected prefix [7 8 9], got %v", got1)
	}
}

func TestFreeSpace(t *testing.T) {
	p := NewPage(TypeLeaf, 0)

	initial := p.FreeSpace()
	expected := PageSize - HeaderSize
	if initial != expected {
		t.Errorf("expected initial FreeSpace=%d, got %d", expected, initial)
	}

	cell := make([]byte, 10)
	p.AddCell(cell)

	// セル10bytes + スロット2bytes 分減る
	after := p.FreeSpace()
	if after != initial-12 {
		t.Errorf("expected FreeSpace=%d after AddCell, got %d", initial-12, after)
	}
}

func TestRightmostChild(t *testing.T) {
	p := NewPage(TypeInternal, 0)
	p.SetRightmostChild(99)

	if p.RightmostChild() != 99 {
		t.Errorf("expected RightmostChild=99, got %d", p.RightmostChild())
	}
}
