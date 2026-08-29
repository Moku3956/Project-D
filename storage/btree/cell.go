package btree

import (
	"encoding/binary"

	"github.com/Moku3956/Project-D/types"
)

// encodeCompositeKey は [tableID(4 BE)][type_tag(1)][pk_bytes] を返す。
// type_tag: 0x01=INT, 0x02=VARCHAR
func encodeCompositeKey(tableID uint32, v types.Value) []byte {
	var tag byte
	var pkBytes []byte
	switch val := v.(type) {
	case types.IntValue:
		tag = 0x01
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(val.V))
		pkBytes = b
	case types.StringValue:
		tag = 0x02
		s := []byte(val.V)
		b := make([]byte, 2+len(s))
		binary.BigEndian.PutUint16(b[0:2], uint16(len(s)))
		copy(b[2:], s)
		pkBytes = b
	}
	buf := make([]byte, 5+len(pkBytes))
	binary.BigEndian.PutUint32(buf[0:4], tableID)
	buf[4] = tag
	copy(buf[5:], pkBytes)
	return buf
}

// decodeCompositeKey は自己記述型の複合キーをデコードし、消費バイト数nを返す。
func decodeCompositeKey(b []byte) (tableID uint32, pk types.Value, n int) {
	tableID = binary.BigEndian.Uint32(b[0:4])
	tag := b[4]
	switch tag {
	case 0x01: // INT
		v := int64(binary.BigEndian.Uint64(b[5:13]))
		pk = types.IntValue{V: v}
		n = 13
	case 0x02: // VARCHAR
		strLen := int(binary.BigEndian.Uint16(b[5:7]))
		pk = types.StringValue{V: string(b[7 : 7+strLen])}
		n = 7 + strLen
	}
	return
}

// cellTableID はセルの先頭4バイトからtableIDを読む。葉/内部どちらにも使える。
func cellTableID(cell []byte) uint32 {
	return binary.BigEndian.Uint32(cell[0:4])
}

// compareCompositeKeys はtableIDを先に、同じなら値で比較する。
func compareCompositeKeys(aID uint32, aPK types.Value, bID uint32, bPK types.Value) int {
	if aID != bID {
		if aID < bID {
			return -1
		}
		return 1
	}
	return compareValues(aPK, bPK)
}

func compareValues(a, b types.Value) int {
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
// フォーマット: [compositeKey][NULLビットマップ][offset配列(各2bytes)][カラムデータ...]
func encodeLeafCell(tableID uint32, key types.Value, row types.Row, schema *types.Schema) []byte {
	keyBytes := encodeCompositeKey(tableID, key)

	colData := make([][]byte, len(schema.Columns))
	for i := range schema.Columns {
		colData[i] = encodeValue(row.Values[i])
	}

	n := len(schema.Columns)
	bitmapSize := 1
	if n > 8 {
		bitmapSize = 2
	}
	bitmap := make([]byte, bitmapSize)
	for i, v := range row.Values {
		if types.KindOf(v) == types.KindNull {
			bitmap[i/8] |= 1 << uint(i%8)
		}
	}

	offsets := make([]byte, n*2)
	pos := uint16(len(keyBytes) + bitmapSize + n*2)
	for i, d := range colData {
		binary.BigEndian.PutUint16(offsets[i*2:], pos)
		pos += uint16(len(d))
	}

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

func encodeValue(v types.Value) []byte {
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

// decodeLeafCell は葉ノードのセルをデコードしてtableID・PK・Rowを返す。
func decodeLeafCell(cell []byte, schema *types.Schema) (tableID uint32, pk types.Value, row types.Row) {
	tableID, pk, keyLen := decodeCompositeKey(cell)

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
		if bitmap[i/8]&(1<<uint(i%8)) != 0 {
			values[i] = types.NullValue{}
			continue
		}
		off := int(binary.BigEndian.Uint16(cell[offsetsStart+i*2:])) - (keyLen + bitmapSize + n*2) + dataStart
		v, _ := decodeValue(cell[off:], col.Type)
		values[i] = v
	}

	row = types.Row{Values: values}
	return
}

// encodeInternalCell は内部ノードのセルをエンコードする。
// フォーマット: [compositeKey][childPageID 4bytes]
func encodeInternalCell(tableID uint32, key types.Value, childPageID uint32) []byte {
	keyBytes := encodeCompositeKey(tableID, key)
	buf := make([]byte, len(keyBytes)+4)
	copy(buf, keyBytes)
	binary.BigEndian.PutUint32(buf[len(keyBytes):], childPageID)
	return buf
}

// decodeInternalCell は内部ノードのセルをデコードする。
func decodeInternalCell(cell []byte) (tableID uint32, key types.Value, childID uint32) {
	tableID, key, n := decodeCompositeKey(cell)
	childID = binary.BigEndian.Uint32(cell[n : n+4])
	return
}

// sortCells は複合キー順にセルをソートする（バブルソート）。
// 葉/内部どちらのセルも先頭が複合キーなので共通で使える。
func sortCells(cells [][]byte) {
	for i := 0; i < len(cells); i++ {
		for j := 0; j < len(cells)-1-i; j++ {
			tidA, pkA, _ := decodeCompositeKey(cells[j])
			tidB, pkB, _ := decodeCompositeKey(cells[j+1])
			if compareCompositeKeys(tidA, pkA, tidB, pkB) > 0 {
				cells[j], cells[j+1] = cells[j+1], cells[j]
			}
		}
	}
}
