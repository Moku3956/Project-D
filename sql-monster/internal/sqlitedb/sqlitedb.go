// Package sqlitedb はsql-monsterのDBアクセスを提供する。
//
// Project-D本体のSQL言語には算術演算子(+/-/*//)がまだ実装されておらず、
// 「UPDATE monsters SET hp = hp - 50」のような攻撃SQLが書けない
// (project_issues.md「算術演算子が言語に存在しない」参照)。この制約を
// 回避するため、sql-monsterのDBエンジンをいったんSQLiteに差し替える。
// Project-D本体への算術演算子の実装は別途行う。
//
// client.DB(Project-D本体の公開API)とほぼ同じメソッド形状にしてあるのは、
// internal/game・internal/apiの呼び出し側をほぼ変更せずに済ませるため。
package sqlitedb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // database/sql に "sqlite" ドライバを登録する

	"github.com/Moku3956/Project-D/types"
)

// ScanKind はクエリがインデックス(PK)を使って絞り込めたか、全件走査だったかを表す。
// spec.mdの「狙いが精密なクエリならクリティカルヒットになる」の判定材料。
type ScanKind int

const (
	// ScanNone はスキャンを伴わない文(INSERT/CREATE/DROP等)を表す。
	ScanNone ScanKind = iota
	// ScanIndex はSQLiteがインデックス(PKやUNIQUE)を使って絞り込めたことを表す。
	ScanIndex
	// ScanSequential は1つでもテーブル全体を走査するSCANが含まれることを表す。
	ScanSequential
)

// Result はSQL実行の結果。
type Result struct {
	Rows         []types.Row
	AffectedRows int
	Schema       *types.Schema
	Scan         ScanKind
}

// DB はSQLiteファイルへのハンドル。
type DB struct {
	sqlDB *sql.DB
}

// Open はdirにある(なければ新規作成する)SQLiteファイルを開く。
func Open(dir string) (*DB, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "sqlite.db")
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}
	return &DB{sqlDB: sqlDB}, nil
}

// Close はDBファイルのハンドルを閉じる。
func (db *DB) Close() error { return db.sqlDB.Close() }

// Exec はSQLを1文実行する(自動コミット)。
func (db *DB) Exec(sqlText string) (*Result, error) {
	return execAny(db.sqlDB, sqlText)
}

// Begin は明示的なトランザクションを開始する。
func (db *DB) Begin() (*Tx, error) {
	tx, err := db.sqlDB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx}, nil
}

// Tx はBeginで開始した、呼び出し元が寿命を管理するトランザクション。
type Tx struct {
	tx *sql.Tx
}

// Exec はこのトランザクションの中でSQLを1文実行する。コミット/ロールバックはしない。
func (t *Tx) Exec(sqlText string) (*Result, error) {
	return execAny(t.tx, sqlText)
}

// Commit はこのトランザクションをコミットする。
func (t *Tx) Commit() error { return t.tx.Commit() }

// Rollback はこのトランザクションをロールバックする。
func (t *Tx) Rollback() error { return t.tx.Rollback() }

// querier はsql.DBとsql.Txの両方が満たすインターフェース。execAnyをどちらにも使えるようにする。
type querier interface {
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)
}

// execAny はSQLの種類(SELECT系かどうか)を見て、Query/Execを使い分ける。
func execAny(q querier, sqlText string) (*Result, error) {
	if isSelectLike(sqlText) {
		return runQuery(q, sqlText)
	}
	return runExec(q, sqlText)
}

// isSelectLike は行を返す文(SELECT)かどうかを、先頭の単語で判定する。
func isSelectLike(sqlText string) bool {
	trimmed := strings.TrimSpace(sqlText)
	return len(trimmed) >= 6 && strings.EqualFold(trimmed[:6], "SELECT")
}

func runQuery(q querier, sqlText string) (*Result, error) {
	rows, err := q.Query(sqlText)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	schema := &types.Schema{Columns: make([]types.Column, len(cols))}
	for i, c := range cols {
		schema.Columns[i] = types.Column{Name: c}
	}

	var result []types.Row
	for rows.Next() {
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := types.Row{Values: make([]types.Value, len(cols))}
		for i, v := range raw {
			row.Values[i] = toValue(v)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	scan, err := explainScanKind(q, sqlText)
	if err != nil {
		// EXPLAIN自体の失敗でクエリ結果を無駄にしたくないので、ScanNoneにフォールバックする
		scan = ScanNone
	}

	return &Result{Rows: result, AffectedRows: len(result), Schema: schema, Scan: scan}, nil
}

func runExec(q querier, sqlText string) (*Result, error) {
	res, err := q.Exec(sqlText)
	if err != nil {
		return nil, fmt.Errorf("exec failed: %w", err)
	}
	affected, _ := res.RowsAffected() // CREATE/DROP等は非対応なことがあるので無視してよい

	scan := ScanNone
	if isUpdateOrDelete(sqlText) {
		if s, err := explainScanKind(q, sqlText); err == nil {
			scan = s
		}
	}

	return &Result{AffectedRows: int(affected), Scan: scan}, nil
}

func isUpdateOrDelete(sqlText string) bool {
	trimmed := strings.ToUpper(strings.TrimSpace(sqlText))
	return strings.HasPrefix(trimmed, "UPDATE") || strings.HasPrefix(trimmed, "DELETE")
}

// explainScanKind は "EXPLAIN QUERY PLAN <sql>" の結果を見て、インデックスを使えたか判定する。
// SQLiteは絞り込みにインデックス/PKを使えると "SEARCH ..."、使えないと "SCAN ..." を返す。
func explainScanKind(q querier, sqlText string) (ScanKind, error) {
	rows, err := q.Query("EXPLAIN QUERY PLAN " + sqlText)
	if err != nil {
		return ScanNone, err
	}
	defer rows.Close() //nolint:errcheck

	cols, err := rows.Columns()
	if err != nil {
		return ScanNone, err
	}

	hasIndex, hasSeq := false, false
	for rows.Next() {
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return ScanNone, err
		}
		detail := fmt.Sprintf("%v", raw[len(raw)-1]) // detail列は常に最後
		upper := strings.ToUpper(detail)
		switch {
		case strings.Contains(upper, "SEARCH"):
			hasIndex = true
		case strings.Contains(upper, "SCAN"):
			hasSeq = true
		}
	}
	if err := rows.Err(); err != nil {
		return ScanNone, err
	}

	switch {
	case hasSeq:
		return ScanSequential, nil
	case hasIndex:
		return ScanIndex, nil
	default:
		return ScanNone, nil
	}
}

// toValue はdatabase/sqlが返すGoの値をtypes.Valueに変換する。
func toValue(v any) types.Value {
	switch val := v.(type) {
	case nil:
		return types.NullValue{}
	case int64:
		return types.IntValue{V: val}
	case float64:
		return types.IntValue{V: int64(val)}
	case string:
		return types.StringValue{V: val}
	case []byte:
		return types.StringValue{V: string(val)}
	case bool:
		return types.BoolValue{V: val}
	default:
		return types.StringValue{V: fmt.Sprintf("%v", val)}
	}
}
