package btree

import (
	"testing"

	"github.com/Moku3956/Project-D/types"
)

// ---- 正常系 ----

// IntValue をエンコードしてデコードすると元の値に戻ることを確認する。
func TestEncodeDecodeKeyInt(t *testing.T) {
	original := types.IntValue{V: 42}
	b := encodeKey(original)
	got, n := decodeKey(b, types.KindIntType)

	if n != 8 {
		t.Errorf("expected n=8, got %d", n)
	}
	v, ok := got.(types.IntValue)
	if !ok || v.V != 42 {
		t.Errorf("expected IntValue{42}, got %v", got)
	}
}

// StringValue をエンコードしてデコードすると元の値に戻ることを確認する。
func TestEncodeDecodeKeyString(t *testing.T) {
	original := types.StringValue{V: "hello"}
	b := encodeKey(original)
	got, n := decodeKey(b, types.KindVarcharType)

	if n != 2+5 {
		t.Errorf("expected n=7, got %d", n)
	}
	v, ok := got.(types.StringValue)
	if !ok || v.V != "hello" {
		t.Errorf("expected StringValue{hello}, got %v", got)
	}
}

// compareKeys が Int の大小を正しく返すことを確認する。
func TestCompareKeysInt(t *testing.T) {
	a := types.IntValue{V: 1}
	b := types.IntValue{V: 2}
	c := types.IntValue{V: 1}

	if compareKeys(a, b) >= 0 {
		t.Error("expected a < b")
	}
	if compareKeys(b, a) <= 0 {
		t.Error("expected b > a")
	}
	if compareKeys(a, c) != 0 {
		t.Error("expected a == c")
	}
}

// compareKeys が String を辞書順で正しく比較することを確認する。
func TestCompareKeysString(t *testing.T) {
	a := types.StringValue{V: "apple"}
	b := types.StringValue{V: "banana"}

	if compareKeys(a, b) >= 0 {
		t.Error("expected apple < banana")
	}
	if compareKeys(b, a) <= 0 {
		t.Error("expected banana > apple")
	}
}

// 通常のRowをエンコードしてデコードすると元のkey・Rowに戻ることを確認する。
func TestEncodeDecodeLeafCell(t *testing.T) {
	schema := testSchema()
	key := types.IntValue{V: 1}
	row := types.Row{Values: []types.Value{
		types.IntValue{V: 1},
		types.StringValue{V: "alice"},
	}}

	cell := encodeLeafCell(key, row, schema)
	gotKey, gotRow := decodeLeafCell(cell, schema)

	k, ok := gotKey.(types.IntValue)
	if !ok || k.V != 1 {
		t.Errorf("expected key=1, got %v", gotKey)
	}
	name, ok := gotRow.Values[1].(types.StringValue)
	if !ok || name.V != "alice" {
		t.Errorf("expected name=alice, got %v", gotRow.Values[1])
	}
}

// NULLを含むRowが正しくエンコード・デコードされることを確認する。
func TestEncodeDecodeLeafCellNull(t *testing.T) {
	schema := testSchema()
	key := types.IntValue{V: 2}
	row := types.Row{Values: []types.Value{
		types.IntValue{V: 2},
		types.NullValue{},
	}}

	cell := encodeLeafCell(key, row, schema)
	_, gotRow := decodeLeafCell(cell, schema)

	if types.KindOf(gotRow.Values[1]) != types.KindNull {
		t.Errorf("expected NullValue, got %v", gotRow.Values[1])
	}
}

// encodeInternalCell / decodeInternalCell でkey・childPageIDが元に戻ることを確認する。
func TestEncodeDecodeInternalCell(t *testing.T) {
	key := types.IntValue{V: 10}
	childID := uint32(99)

	cell := encodeInternalCell(key, childID)
	gotKey, gotChild := decodeInternalCell(cell, types.KindIntType)

	k, ok := gotKey.(types.IntValue)
	if !ok || k.V != 10 {
		t.Errorf("expected key=10, got %v", gotKey)
	}
	if gotChild != childID {
		t.Errorf("expected childID=99, got %d", gotChild)
	}
}

// ---- 異常系 ----

// BoolValue を encodeKey に渡すと nil が返ることを確認する。
func TestEncodeKeyBool(t *testing.T) {
	b := encodeKey(types.BoolValue{V: true})
	if b != nil {
		t.Errorf("expected nil, got %v", b)
	}
}

// NullValue を encodeKey に渡すと nil が返ることを確認する。
func TestEncodeKeyNull(t *testing.T) {
	b := encodeKey(types.NullValue{})
	if b != nil {
		t.Errorf("expected nil, got %v", b)
	}
}

// 型が異なる2値を compareKeys に渡すと panic することを確認する。
func TestCompareKeysDifferentTypes(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for mismatched types")
		}
	}()
	compareKeys(types.IntValue{V: 1}, types.StringValue{V: "a"})
}
