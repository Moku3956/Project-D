package btree

import (
	"encoding/binary"

	"github.com/Moku3956/Project-D/types"
)

// encodeKey はPKのValueをバイト列にエンコードする。
func encodeKey(v types.Value) []byte {
	switch val := v.(type) {
	case types.IntValue:
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(val.V))
		return b
	case types.StringValue:
		s := []byte(val.V)
		b := make([]byte, 2+len(s))
		binary.BigEndian.PutUint16(b[0:2], uint16(len(s)))
		copy(b[2:], s)
		return b
	default:
		return nil
	}
}

// decodeKey はバイト列をPKのValueにデコードする。
func decodeKey(b []byte, kind types.DataTypeKind) (types.Value, int) {
	switch kind {
	case types.KindIntType:
		v := int64(binary.BigEndian.Uint64(b[0:8]))
		return types.IntValue{V: v}, 8
	case types.KindVarcharType:
		length := int(binary.BigEndian.Uint16(b[0:2]))
		return types.StringValue{V: string(b[2 : 2+length])}, 2 + length
	default:
		return types.NullValue{}, 0
	}
}

// compareKeys はa < b なら負、a == b なら0、a > b なら正を返す。
func compareKeys(a, b types.Value) int {
	switch av := a.(type) {
	case types.IntValue:
		bv := b.(types.IntValue)
		if av.V < bv.V {
			return -1
		} else if av.V > bv.V {
			return 1
		}
		return 0
	case types.StringValue:
		bv := b.(types.StringValue)
		if av.V < bv.V {
			return -1
		} else if av.V > bv.V {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// encodeLeafCell は葉ノードのセルをエンコードする。
// フォーマット: [key][NULLビットマップ][offset配列(各2bytes)][カラムデータ...]
func encodeLeafCell(key types.Value, row types.Row, schema *types.Schema) []byte {
	keyBytes := encodeKey(key)

	// カラムデータをエンコード
	colData := make([][]byte, len(schema.Columns))
	for i, col := range schema.Columns {
		if col.PrimaryKey {
			colData[i] = keyBytes
			continue
		}
		colData[i] = encodeValue(row.Values[i], col.Type)
	}

	// NULLビットマップ
	n := len(schema.Columns)
	bitmapSize := 1
	if n > 8 {
		bitmapSize = 2
	}
	bitmap := make([]byte, bitmapSize)
	for i, v := range row.Values {
		if types.KindOf(v) == types.KindNull {
			byteIdx := i / 8
			bitIdx := uint(i % 8)
			bitmap[byteIdx] |= 1 << bitIdx
		}
	}

	// オフセット配列
	offsets := make([]byte, n*2)
	pos := uint16(len(keyBytes) + bitmapSize + n*2)
	for i, d := range colData {
		binary.BigEndian.PutUint16(offsets[i*2:], pos)
		pos += uint16(len(d))
	}

	// 結合
	total := len(keyBytes) + bitmapSize + len(offsets)
	for _, d := range colData {
		total += len(d)
	}
	buf := make([]byte, total)
	cur := 0
	copy(buf[cur:], keyBytes)
	cur += len(keyBytes)
	copy(buf[cur:], bitmap)
	cur += bitmapSize
	copy(buf[cur:], offsets)
	cur += len(offsets)
	for _, d := range colData {
		copy(buf[cur:], d)
		cur += len(d)
	}
	return buf
}

func encodeValue(v types.Value, dt types.DataType) []byte {
	switch val := v.(type) {
	case types.IntValue:
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(val.V))
		return b
	case types.StringValue:
		s := []byte(val.V)
		b := make([]byte, 2+len(s))
		binary.BigEndian.PutUint16(b[0:2], uint16(len(s)))
		copy(b[2:], s)
		return b
	case types.BoolValue:
		if val.V {
			return []byte{1}
		}
		return []byte{0}
	default:
		return []byte{}
	}
}

func decodeValue(b []byte, dt types.DataType) (types.Value, int) {
	switch dt.Kind {
	case types.KindIntType:
		v := int64(binary.BigEndian.Uint64(b[0:8]))
		return types.IntValue{V: v}, 8
	case types.KindVarcharType:
		length := int(binary.BigEndian.Uint16(b[0:2]))
		return types.StringValue{V: string(b[2 : 2+length])}, 2 + length
	case types.KindBoolType:
		return types.BoolValue{V: b[0] == 1}, 1
	default:
		return types.NullValue{}, 0
	}
}

// decodeLeafCell は葉ノードのセルをデコードしてRowを返す。
func decodeLeafCell(cell []byte, schema *types.Schema) (types.Value, types.Row) {
	pkIdx := schema.PrimaryKeyIndex()
	pkKind := schema.Columns[pkIdx].Type.Kind
	key, keyLen := decodeKey(cell, pkKind)

	n := len(schema.Columns)
	bitmapSize := 1
	if n > 8 {
		bitmapSize = 2
	}

	bitmap := cell[keyLen : keyLen+bitmapSize]
	offsetsStart := keyLen + bitmapSize
	dataStart := offsetsStart + n*2

	values := make([]types.Value, n)
	for i, col := range schema.Columns {
		byteIdx := i / 8
		bitIdx := uint(i % 8)
		if bitmap[byteIdx]&(1<<bitIdx) != 0 {
			values[i] = types.NullValue{}
			continue
		}
		off := int(binary.BigEndian.Uint16(cell[offsetsStart+i*2:])) - (keyLen + bitmapSize + n*2) + dataStart
		v, _ := decodeValue(cell[off:], col.Type)
		values[i] = v
	}

	return key, types.Row{Values: values}
}

// encodeInternalCell は内部ノードのセルをエンコードする。
// フォーマット: [key][childPageID 4bytes]
func encodeInternalCell(key types.Value, childPageID uint32) []byte {
	keyBytes := encodeKey(key)
	buf := make([]byte, len(keyBytes)+4)
	copy(buf, keyBytes)
	binary.BigEndian.PutUint32(buf[len(keyBytes):], childPageID)
	return buf
}

// decodeInternalCell は内部ノードのセルをデコードする。
func decodeInternalCell(cell []byte, keyKind types.DataTypeKind) (types.Value, uint32) {
	key, keyLen := decodeKey(cell, keyKind)
	childID := binary.BigEndian.Uint32(cell[keyLen : keyLen+4])
	return key, childID
}
