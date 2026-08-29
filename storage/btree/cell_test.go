package btree

import (
	"testing"

	"github.com/Moku3956/Project-D/types"
)

const cellTestTableID = uint32(5)

// ---- 正常系 ----

func TestEncodeDecodeCompositeKeyInt(t *testing.T) {
	original := types.IntValue{V: 42}
	b := encodeCompositeKey(cellTestTableID, original)

	tid, got, n := decodeCompositeKey(b)
	if tid != cellTestTableID {
		t.Errorf("tableID = %d, want %d", tid, cellTestTableID)
	}
	if n != 13 { // 4(tableID) + 1(tag) + 8(int64)
		t.Errorf("n = %d, want 13", n)
	}
	v, ok := got.(types.IntValue)
	if !ok || v.V != 42 {
		t.Errorf("pk = %v, want IntValue{42}", got)
	}
}

func TestEncodeDecodeCompositeKeyString(t *testing.T) {
	original := types.StringValue{V: "hello"}
	b := encodeCompositeKey(cellTestTableID, original)

	tid, got, n := decodeCompositeKey(b)
	if tid != cellTestTableID {
		t.Errorf("tableID = %d, want %d", tid, cellTestTableID)
	}
	if n != 4+1+2+5 { // 4(tableID) + 1(tag) + 2(len) + 5(bytes)
		t.Errorf("n = %d, want %d", n, 4+1+2+5)
	}
	v, ok := got.(types.StringValue)
	if !ok || v.V != "hello" {
		t.Errorf("pk = %v, want StringValue{hello}", got)
	}
}

func TestCompareCompositeKeysDifferentTable(t *testing.T) {
	if compareCompositeKeys(1, types.IntValue{V: 999}, 2, types.IntValue{V: 1}) >= 0 {
		t.Error("tableID=1 should be less than tableID=2 regardless of pk")
	}
}

func TestCompareValuesInt(t *testing.T) {
	a := types.IntValue{V: 1}
	b := types.IntValue{V: 2}
	c := types.IntValue{V: 1}

	if compareValues(a, b) >= 0 {
		t.Error("expected a < b")
	}
	if compareValues(b, a) <= 0 {
		t.Error("expected b > a")
	}
	if compareValues(a, c) != 0 {
		t.Error("expected a == c")
	}
}

func TestCompareValuesString(t *testing.T) {
	a := types.StringValue{V: "apple"}
	b := types.StringValue{V: "banana"}

	if compareValues(a, b) >= 0 {
		t.Error("expected apple < banana")
	}
	if compareValues(b, a) <= 0 {
		t.Error("expected banana > apple")
	}
}

func TestEncodeDecodeLeafCell(t *testing.T) {
	schema := testSchema()
	key := types.IntValue{V: 1}
	row := types.Row{Values: []types.Value{
		types.IntValue{V: 1},
		types.StringValue{V: "alice"},
	}}

	cell := encodeLeafCell(cellTestTableID, key, row, schema)
	gotTID, gotKey, gotRow := decodeLeafCell(cell, schema)

	if gotTID != cellTestTableID {
		t.Errorf("tableID = %d, want %d", gotTID, cellTestTableID)
	}
	k, ok := gotKey.(types.IntValue)
	if !ok || k.V != 1 {
		t.Errorf("pk = %v, want 1", gotKey)
	}
	name, ok := gotRow.Values[1].(types.StringValue)
	if !ok || name.V != "alice" {
		t.Errorf("name = %v, want alice", gotRow.Values[1])
	}
}

func TestEncodeDecodeLeafCellNull(t *testing.T) {
	schema := testSchema()
	key := types.IntValue{V: 2}
	row := types.Row{Values: []types.Value{
		types.IntValue{V: 2},
		types.NullValue{},
	}}

	cell := encodeLeafCell(cellTestTableID, key, row, schema)
	_, _, gotRow := decodeLeafCell(cell, schema)

	if types.KindOf(gotRow.Values[1]) != types.KindNull {
		t.Errorf("expected NullValue, got %v", gotRow.Values[1])
	}
}

func TestEncodeDecodeInternalCell(t *testing.T) {
	key := types.IntValue{V: 10}
	childID := uint32(99)

	cell := encodeInternalCell(cellTestTableID, key, childID)
	gotTID, gotKey, gotChild := decodeInternalCell(cell)

	if gotTID != cellTestTableID {
		t.Errorf("tableID = %d, want %d", gotTID, cellTestTableID)
	}
	k, ok := gotKey.(types.IntValue)
	if !ok || k.V != 10 {
		t.Errorf("key = %v, want 10", gotKey)
	}
	if gotChild != childID {
		t.Errorf("childID = %d, want %d", gotChild, childID)
	}
}

func TestCellTableID(t *testing.T) {
	cell := encodeCompositeKey(42, types.IntValue{V: 1})
	if cellTableID(cell) != 42 {
		t.Errorf("cellTableID = %d, want 42", cellTableID(cell))
	}
}
