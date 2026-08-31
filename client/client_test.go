package client

import (
	"testing"

	"github.com/Moku3956/Project-D/types"
)

// ---- 正常系 ----

func TestOpenCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer db.Close() //nolint:errcheck
}

func TestExecCreateTableAndInsertAndSelect(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer db.Close() //nolint:errcheck

	if _, err := db.Exec("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))"); err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}
	if _, err := db.Exec("INSERT INTO users VALUES (1, 'Alice')"); err != nil {
		t.Fatalf("INSERT error: %v", err)
	}

	result, err := db.Exec("SELECT * FROM users")
	if err != nil {
		t.Fatalf("SELECT error: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("レコード数 = %d, want 1", len(result.Rows))
	}
}

func TestExecReportsScanKind(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer db.Close() //nolint:errcheck

	if _, err := db.Exec("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))"); err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}
	if _, err := db.Exec("INSERT INTO users VALUES (1, 'Alice')"); err != nil {
		t.Fatalf("INSERT error: %v", err)
	}
	if _, err := db.Exec("INSERT INTO users VALUES (2, 'Bob')"); err != nil {
		t.Fatalf("INSERT error: %v", err)
	}

	indexResult, err := db.Exec("SELECT * FROM users WHERE id = 1")
	if err != nil {
		t.Fatalf("SELECT error: %v", err)
	}
	if indexResult.Scan != ScanIndex {
		t.Errorf("Scan = %v, want ScanIndex", indexResult.Scan)
	}

	seqResult, err := db.Exec("SELECT * FROM users WHERE name = 'Bob'")
	if err != nil {
		t.Fatalf("SELECT error: %v", err)
	}
	if seqResult.Scan != ScanSequential {
		t.Errorf("Scan = %v, want ScanSequential", seqResult.Scan)
	}
}

// Beginで開始したトランザクションでExecした変更が、Commit後に反映されていることを確認する。
func TestTxCommit(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer db.Close() //nolint:errcheck

	if _, err := db.Exec("CREATE TABLE monsters (id INT PRIMARY KEY, hp INT)"); err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}
	if _, err := db.Exec("INSERT INTO monsters VALUES (1, 100)"); err != nil {
		t.Fatalf("INSERT error: %v", err)
	}

	// SET句は算術式(hp - 50 のような)をまだサポートしていないため、リテラル代入で確認する。
	// 詳細はproject_issues.mdの「UPDATEのSET句が算術式に対応していない」を参照。
	tx := db.Begin()
	if _, err := tx.Exec("UPDATE monsters SET hp = 50 WHERE id = 1"); err != nil {
		t.Fatalf("Exec error: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit error: %v", err)
	}

	result, err := db.Exec("SELECT * FROM monsters WHERE id = 1")
	if err != nil {
		t.Fatalf("SELECT error: %v", err)
	}
	hp := result.Rows[0].Values[1]
	if hp != (types.IntValue{V: 50}) {
		t.Errorf("hp = %v, want 50", hp)
	}
}

func TestTxRollbackDiscardsChange(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer db.Close() //nolint:errcheck

	if _, err := db.Exec("CREATE TABLE monsters (id INT PRIMARY KEY, hp INT)"); err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}
	if _, err := db.Exec("INSERT INTO monsters VALUES (1, 100)"); err != nil {
		t.Fatalf("INSERT error: %v", err)
	}

	tx := db.Begin()
	if _, err := tx.Exec("UPDATE monsters SET hp = 50 WHERE id = 1"); err != nil {
		t.Fatalf("Exec error: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback error: %v", err)
	}

	result, err := db.Exec("SELECT * FROM monsters WHERE id = 1")
	if err != nil {
		t.Fatalf("SELECT error: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("レコード数 = %d, want 1", len(result.Rows))
	}
	hp := result.Rows[0].Values[1]
	if hp != (types.IntValue{V: 100}) {
		t.Errorf("hp = %v, want 100(Rollbackにより変更前の値のまま)", hp)
	}
}

// ---- 異常系 ----

func TestExecParseError(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer db.Close() //nolint:errcheck

	if _, err := db.Exec("SELECT FROM"); err == nil {
		t.Fatal("パースエラーが期待されたがnil")
	}
}

func TestOpenRecoversAfterReopen(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	if _, err := db.Exec("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))"); err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}
	if _, err := db.Exec("INSERT INTO users VALUES (1, 'Alice')"); err != nil {
		t.Fatalf("INSERT error: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}

	db2, err := Open(dir)
	if err != nil {
		t.Fatalf("Open (再オープン) error: %v", err)
	}
	defer db2.Close() //nolint:errcheck

	result, err := db2.Exec("SELECT * FROM users")
	if err != nil {
		t.Fatalf("SELECT error: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("再オープン後のレコード数 = %d, want 1", len(result.Rows))
	}
}
