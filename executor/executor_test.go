package executor

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/Moku3956/Project-D/sql/parser"
	"github.com/Moku3956/Project-D/sql/planner"
	"github.com/Moku3956/Project-D/storage/wal"
	"github.com/Moku3956/Project-D/txn"
	"github.com/Moku3956/Project-D/types"
)

// ---- モック ----

type mockCatalog struct {
	schemas     map[string]*types.Schema
	nextTableID uint32
}

func newMockCatalog() *mockCatalog {
	return &mockCatalog{schemas: make(map[string]*types.Schema)}
}

func (c *mockCatalog) GetSchema(table string) (*types.Schema, error) {
	s, ok := c.schemas[table]
	if !ok {
		return nil, fmt.Errorf("table %q not found", table)
	}
	return s, nil
}

func (c *mockCatalog) TableExists(table string) bool {
	_, ok := c.schemas[table]
	return ok
}

func (c *mockCatalog) CreateTable(schema types.Schema) error {
	c.nextTableID++
	schema.TableID = c.nextTableID
	c.schemas[schema.TableName] = &schema
	return nil
}

func (c *mockCatalog) DropTable(table string) error {
	if _, ok := c.schemas[table]; !ok {
		return fmt.Errorf("table %q not found", table)
	}
	delete(c.schemas, table)
	return nil
}

type mockRepo struct {
	schemas map[string]*types.Schema
	data    map[string][]types.Row
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		schemas: make(map[string]*types.Schema),
		data:    make(map[string][]types.Row),
	}
}

func (r *mockRepo) OpenTable(table string, schema *types.Schema) error {
	r.schemas[table] = schema
	if _, ok := r.data[table]; !ok {
		r.data[table] = []types.Row{}
	}
	return nil
}

func (r *mockRepo) Scan(table string) ([]types.Row, error) {
	rows, ok := r.data[table]
	if !ok {
		return nil, fmt.Errorf("table %q not found", table)
	}
	return rows, nil
}

func (r *mockRepo) FindByPK(table string, pk types.Value) (*types.Row, error) {
	schema, ok := r.schemas[table]
	if !ok {
		return nil, fmt.Errorf("table %q not found", table)
	}
	pkIdx := schema.PrimaryKeyIndex()
	for _, row := range r.data[table] {
		if compareValues(row.Values[pkIdx], pk) == 0 {
			cp := row
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *mockRepo) Insert(table string, row types.Row) error {
	r.data[table] = append(r.data[table], row)
	return nil
}

func (r *mockRepo) Update(table string, pk types.Value, row types.Row) error {
	schema, ok := r.schemas[table]
	if !ok {
		return fmt.Errorf("table %q not found", table)
	}
	pkIdx := schema.PrimaryKeyIndex()
	for i, existing := range r.data[table] {
		if compareValues(existing.Values[pkIdx], pk) == 0 {
			r.data[table][i] = row
			return nil
		}
	}
	return fmt.Errorf("record not found")
}

func (r *mockRepo) Delete(table string, pk types.Value) error {
	schema, ok := r.schemas[table]
	if !ok {
		return fmt.Errorf("table %q not found", table)
	}
	pkIdx := schema.PrimaryKeyIndex()
	for i, row := range r.data[table] {
		if compareValues(row.Values[pkIdx], pk) == 0 {
			r.data[table] = append(r.data[table][:i], r.data[table][i+1:]...)
			return nil
		}
	}
	return nil
}

// ---- ヘルパー ----

// setup はテスト用のカタログ・リポジトリ・Engineを生成する。
func setup(t *testing.T) (*mockCatalog, *mockRepo, *Engine) {
	t.Helper()
	cat := newMockCatalog()
	repo := newMockRepo()
	wm, err := wal.NewWALManager(filepath.Join(t.TempDir(), "test.wal"))
	if err != nil {
		t.Fatalf("NewWALManager error: %v", err)
	}
	eng := NewEngine(repo, cat, txn.NewManager(wm))
	return cat, repo, eng
}

// run はSQL文字列をParse→Plan→Executeして結果を返す。エラーがあればt.Fatalする。
func run(t *testing.T, cat *mockCatalog, repo *mockRepo, eng *Engine, sql string) *Result {
	t.Helper()
	stmt, err := parser.Parse(sql)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	pl := planner.NewPlanner(cat)
	node, err := pl.Plan(stmt)
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	result, err := eng.Execute(node)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	return result
}

// runErr はSQL文字列をParse→Plan→Executeしてエラーを返す。
func runErr(t *testing.T, cat *mockCatalog, repo *mockRepo, eng *Engine, sql string) error {
	t.Helper()
	stmt, err := parser.Parse(sql)
	if err != nil {
		return err
	}
	pl := planner.NewPlanner(cat)
	node, err := pl.Plan(stmt)
	if err != nil {
		return err
	}
	_, err = eng.Execute(node)
	return err
}

// ---- 正常系 ----

func TestExecuteCreateTable(t *testing.T) {
	cat, repo, eng := setup(t)
	run(t, cat, repo, eng, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")

	if !cat.TableExists("users") {
		t.Error("カタログにusersが登録されていない")
	}
	if _, ok := repo.schemas["users"]; !ok {
		t.Error("リポジトリにusersが登録されていない")
	}
}

func TestExecuteInsert(t *testing.T) {
	cat, repo, eng := setup(t)
	run(t, cat, repo, eng, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (1, 'Alice')")

	if len(repo.data["users"]) != 1 {
		t.Fatalf("レコード数 = %d, want 1", len(repo.data["users"]))
	}
}

func TestExecuteSelect(t *testing.T) {
	cat, repo, eng := setup(t)
	run(t, cat, repo, eng, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (1, 'Alice')")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (2, 'Bob')")

	result := run(t, cat, repo, eng, "SELECT * FROM users")
	if len(result.Rows) != 2 {
		t.Fatalf("レコード数 = %d, want 2", len(result.Rows))
	}
}

func TestExecuteSelectWhere(t *testing.T) {
	cat, repo, eng := setup(t)
	run(t, cat, repo, eng, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (1, 'Alice')")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (2, 'Bob')")

	result := run(t, cat, repo, eng, "SELECT * FROM users WHERE id = 2")
	if len(result.Rows) != 1 {
		t.Fatalf("レコード数 = %d, want 1", len(result.Rows))
	}
	if result.Rows[0].Values[0] != (types.IntValue{V: 2}) {
		t.Errorf("id = %v, want 2", result.Rows[0].Values[0])
	}
}

func TestExecuteSelectIndexScan(t *testing.T) {
	cat, repo, eng := setup(t)
	run(t, cat, repo, eng, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (1, 'Alice')")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (2, 'Bob')")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (3, 'Carol')")

	result := run(t, cat, repo, eng, "SELECT * FROM users WHERE id = 2")
	if len(result.Rows) != 1 {
		t.Fatalf("レコード数 = %d, want 1", len(result.Rows))
	}
	if result.Rows[0].Values[0] != (types.IntValue{V: 2}) {
		t.Errorf("id = %v, want 2", result.Rows[0].Values[0])
	}
}

func TestExecuteUpdate(t *testing.T) {
	cat, repo, eng := setup(t)
	run(t, cat, repo, eng, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (1, 'Alice')")
	run(t, cat, repo, eng, "UPDATE users SET name = 'Bob' WHERE id = 1")

	result := run(t, cat, repo, eng, "SELECT * FROM users")
	if len(result.Rows) != 1 {
		t.Fatalf("レコード数 = %d, want 1", len(result.Rows))
	}
	if result.Rows[0].Values[1] != (types.StringValue{V: "Bob"}) {
		t.Errorf("name = %v, want Bob", result.Rows[0].Values[1])
	}
}

func TestExecuteDelete(t *testing.T) {
	cat, repo, eng := setup(t)
	run(t, cat, repo, eng, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (1, 'Alice')")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (2, 'Bob')")
	run(t, cat, repo, eng, "DELETE FROM users WHERE id = 1")

	result := run(t, cat, repo, eng, "SELECT * FROM users")
	if len(result.Rows) != 1 {
		t.Fatalf("レコード数 = %d, want 1", len(result.Rows))
	}
	if result.Rows[0].Values[0] != (types.IntValue{V: 2}) {
		t.Errorf("id = %v, want 2", result.Rows[0].Values[0])
	}
}

func TestExecuteSelectOrderByLimit(t *testing.T) {
	cat, repo, eng := setup(t)
	run(t, cat, repo, eng, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (1, 'Alice')")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (2, 'Bob')")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (3, 'Carol')")

	result := run(t, cat, repo, eng, "SELECT * FROM users ORDER BY id DESC LIMIT 2")
	if len(result.Rows) != 2 {
		t.Fatalf("レコード数 = %d, want 2", len(result.Rows))
	}
	if result.Rows[0].Values[0] != (types.IntValue{V: 3}) {
		t.Errorf("1件目のid = %v, want 3", result.Rows[0].Values[0])
	}
	if result.Rows[1].Values[0] != (types.IntValue{V: 2}) {
		t.Errorf("2件目のid = %v, want 2", result.Rows[1].Values[0])
	}
}

func TestExecuteDropTable(t *testing.T) {
	cat, repo, eng := setup(t)
	run(t, cat, repo, eng, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	run(t, cat, repo, eng, "DROP TABLE users")

	if cat.TableExists("users") {
		t.Error("カタログからusersが削除されていない")
	}
}

// TestExecuteConcurrentUpdateSerializes は、同じテーブルへの書き込みロックを外部で
// 保持している間、UPDATEがブロックされ、解放後に実行されることを確認する。
func TestExecuteConcurrentUpdateSerializes(t *testing.T) {
	cat, repo, eng := setup(t)
	run(t, cat, repo, eng, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	run(t, cat, repo, eng, "INSERT INTO users VALUES (1, 'Alice')")

	holder := eng.txnMgr.Begin()
	if err := eng.txnMgr.Lock(holder, "users"); err != nil {
		t.Fatalf("Lock error: %v", err)
	}

	stmt, err := parser.Parse("UPDATE users SET name = 'Bob' WHERE id = 1")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	pl := planner.NewPlanner(cat)
	node, err := pl.Plan(stmt)
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := eng.Execute(node)
		done <- err
	}()

	select {
	case <-done:
		t.Fatal("ロックを保持しているのにUPDATEが完了してしまった")
	case <-time.After(200 * time.Millisecond):
	}

	if err := eng.txnMgr.Commit(holder); err != nil {
		t.Fatalf("Commit error: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ロック解放後もUPDATEが完了しなかった")
	}

	result := run(t, cat, repo, eng, "SELECT * FROM users")
	if result.Rows[0].Values[1] != (types.StringValue{V: "Bob"}) {
		t.Errorf("name = %v, want Bob", result.Rows[0].Values[1])
	}
}

func TestExecuteBeginCommitRollback(t *testing.T) {
	cat, repo, eng := setup(t)
	for _, sql := range []string{"BEGIN", "COMMIT", "ROLLBACK"} {
		result := run(t, cat, repo, eng, sql)
		if result == nil {
			t.Errorf("%s: result が nil", sql)
		}
	}
}

// ---- 異常系 ----

func TestExecuteSelectTableNotFound(t *testing.T) {
	cat, repo, eng := setup(t)
	err := runErr(t, cat, repo, eng, "SELECT * FROM users")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
}

func TestExecuteInsertColumnCountMismatch(t *testing.T) {
	cat, repo, eng := setup(t)
	run(t, cat, repo, eng, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	err := runErr(t, cat, repo, eng, "INSERT INTO users VALUES (1)")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
}

func TestExecuteDropTableNotFound(t *testing.T) {
	cat, repo, eng := setup(t)
	err := runErr(t, cat, repo, eng, "DROP TABLE users")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
}
