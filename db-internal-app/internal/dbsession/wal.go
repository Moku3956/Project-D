package dbsession

import (
	"fmt"

	"github.com/Moku3956/Project-D/storage/btree"
	"github.com/Moku3956/Project-D/storage/wal"
	"github.com/Moku3956/Project-D/types"
)

// WalRecord はdb-internal-appのWAL可視化用に、1つのwal.LogRecordを人間に
// 分かる形へデコードしたもの(db-internal-app/docs/spec.md「WAL可視化」参照)。
// RedoDataはページ全体のスナップショットで、これ単体では「このレコードで
// 具体的にどの行が変わったか」は分からないため、WalRecordsが直前の同一ページの
// スナップショットと比較して1行だけを特定する。特定できた場合だけChangedが
// 埋まる(splitで新設されたページのように、複数行が同時に「新規」に見える
// ケースでは特定できずnilのままになる)。
type WalRecord struct {
	LSN        uint64
	TxnID      uint64
	PageID     uint32
	Op         string // "INSERT" | "UPDATE" | "DELETE" | "COMMIT" | "ABORT"
	Table      string // 変更対象テーブル名。COMMIT/ABORT、または特定できなかった場合は空
	ChangeKind string // "added" | "removed" | "changed"。特定できなかった場合は空
	Changed    *types.Row
}

func opName(op wal.Operation) string {
	switch op {
	case wal.OpInsert:
		return "INSERT"
	case wal.OpUpdate:
		return "UPDATE"
	case wal.OpDelete:
		return "DELETE"
	case wal.OpCommit:
		return "COMMIT"
	case wal.OpAbort:
		return "ABORT"
	default:
		return "UNKNOWN"
	}
}

// rowKey はページを跨いだ比較はしない前提で、同一ページ内の行を一意に識別する
// キーを返す(tableName+PK値)。
func rowKey(schema *types.Schema, row types.Row) string {
	return fmt.Sprintf("%s:%v", schema.TableName, row.Values[schema.PrimaryKeyIndex()])
}

// WalRecords はセッションのWALを最初から読み、db-internal-app用にデコードした
// 全レコードをLSN昇順で返す。まだバッファに残っている(Flush前の)レコードも
// 見えるよう、読む前に一度Flushする(ABORTはFlushしないため、直後に呼んでも
// そのままでは見えない。storage/wal/docs/spec.md参照)。
func (s *Session) WalRecords() ([]WalRecord, error) {
	if err := s.wm.Flush(); err != nil {
		return nil, err
	}
	raw, err := s.wm.ReadAll()
	if err != nil {
		return nil, err
	}

	names := s.cat.TableNames()
	schemasByID := make(map[uint32]*types.Schema, len(names))
	schemasByName := make(map[string]*types.Schema, len(names))
	for _, name := range names {
		schema, err := s.cat.GetSchema(name)
		if err != nil {
			return nil, err
		}
		schemasByID[schema.TableID] = schema
		schemasByName[name] = schema
	}

	// ページIDごとに「直前に見たそのページの中身」を覚えておき、次にそのページの
	// レコードが出てきたときと比較して増減・変化した1行を特定する。
	prevRowsByPage := make(map[uint32]map[string]types.Row)

	out := make([]WalRecord, 0, len(raw))
	for _, r := range raw {
		rec := WalRecord{LSN: r.LSN, TxnID: r.TxnID, PageID: r.PageID, Op: opName(r.Op)}

		switch r.Op {
		case wal.OpInsert, wal.OpUpdate, wal.OpDelete:
			rows, rowTables := btree.DecodePageBytes(r.RedoData, schemasByID)
			cur := make(map[string]types.Row, len(rows))
			for i, row := range rows {
				schema := schemasByName[rowTables[i]]
				cur[rowKey(schema, row)] = row
			}
			prev := prevRowsByPage[r.PageID]

			table, kind, changed := diffOneRow(r.Op, prev, cur)
			rec.Table = table
			rec.ChangeKind = kind
			rec.Changed = changed

			prevRowsByPage[r.PageID] = cur
		}

		out = append(out, rec)
	}
	return out, nil
}

// diffOneRow はprev(直前の同一ページの中身)とcur(今回のRedoData)を比較して、
// opの向き(挿入なら増えた行、削除なら消えた行、更新なら内容が変わった行)に
// 一致する候補がちょうど1つだけ見つかった場合にそれを返す。0件・複数件(例えば
// splitで新設されたページのように、複数行が一度に「新規」扱いになるケース)
// では特定を諦めてゼロ値を返す。
func diffOneRow(op wal.Operation, prev, cur map[string]types.Row) (table, kind string, changed *types.Row) {
	switch op {
	case wal.OpInsert:
		return pickSingle(prev, cur, "added")
	case wal.OpDelete:
		return pickSingle(cur, prev, "removed")
	case wal.OpUpdate:
		var candidates []string
		for k, newRow := range cur {
			oldRow, ok := prev[k]
			if ok && !rowsEqual(oldRow, newRow) {
				candidates = append(candidates, k)
			}
		}
		if len(candidates) != 1 {
			return "", "", nil
		}
		row := cur[candidates[0]]
		return tableNameFromKey(candidates[0]), "changed", &row
	default:
		return "", "", nil
	}
}

// pickSingle はbase(比較のベースとなる方)にはなくtarget(比較先)にだけ存在する
// キーがちょうど1つの場合にその行を返す。
func pickSingle(base, target map[string]types.Row, kind string) (table, k string, changed *types.Row) {
	var onlyKey string
	count := 0
	for key := range target {
		if _, ok := base[key]; !ok {
			onlyKey = key
			count++
		}
	}
	if count != 1 {
		return "", "", nil
	}
	row := target[onlyKey]
	return tableNameFromKey(onlyKey), kind, &row
}

func tableNameFromKey(key string) string {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return key[:i]
		}
	}
	return key
}

func rowsEqual(a, b types.Row) bool {
	if len(a.Values) != len(b.Values) {
		return false
	}
	for i := range a.Values {
		if fmt.Sprint(a.Values[i]) != fmt.Sprint(b.Values[i]) {
			return false
		}
	}
	return true
}
