package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/Moku3956/Project-D/catalog"
	"github.com/Moku3956/Project-D/executor"
	"github.com/Moku3956/Project-D/infrastructure"
	"github.com/Moku3956/Project-D/sql/planner"
	"github.com/Moku3956/Project-D/storage/page"
)

// ---- ヘルパー ----

// setup は実物のCatalog・BTreeRepositoryを一時ディレクトリ上に構築し、
// ルーティング済みのServeMuxを返す。HTTPからディスクまでを通す統合テスト用。
func setup(t *testing.T) *http.ServeMux {
	t.Helper()
	dir := t.TempDir()

	dm, err := page.NewDiskManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDiskManager error: %v", err)
	}
	t.Cleanup(func() { dm.Close() }) //nolint:errcheck

	cat, err := catalog.NewCatalog(filepath.Join(dir, "catalog.json"))
	if err != nil {
		t.Fatalf("NewCatalog error: %v", err)
	}

	repo, err := infrastructure.NewBTreeRepository(dm)
	if err != nil {
		t.Fatalf("NewBTreeRepository error: %v", err)
	}

	pl := planner.NewPlanner(cat)
	eng := executor.NewEngine(repo, cat)

	mux := http.NewServeMux()
	NewHandler(pl, eng).RegisterRoutes(mux)
	return mux
}

// post は /query にSQLをPOSTしてレスポンスを返す。
func post(t *testing.T, mux *http.ServeMux, sql string) (*httptest.ResponseRecorder, queryResponse) {
	t.Helper()
	body, _ := json.Marshal(queryRequest{SQL: sql})
	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var resp queryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("レスポンスのデコードに失敗: %v (body=%s)", err, rec.Body.String())
	}
	return rec, resp
}

// postRaw は任意のボディを /query にPOSTする。不正なJSONのテスト用。
func postRaw(t *testing.T, mux *http.ServeMux, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// mustPost はSQLをPOSTしてステータス200でなければ失敗する。前提データの投入用。
func mustPost(t *testing.T, mux *http.ServeMux, sql string) {
	t.Helper()
	rec, resp := post(t, mux, sql)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: ステータス = %d, エラー = %q", sql, rec.Code, resp.Error)
	}
}

// ---- 正常系 ----

func TestHandleHealth(t *testing.T) {
	mux := setup(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ステータス = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("デコードに失敗: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want %q", body["status"], "ok")
	}
}

func TestHandleQueryCreateTable(t *testing.T) {
	mux := setup(t)

	rec, resp := post(t, mux, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	if rec.Code != http.StatusOK {
		t.Fatalf("ステータス = %d, エラー = %q", rec.Code, resp.Error)
	}
	if resp.Error != "" {
		t.Errorf("エラー = %q, want 空", resp.Error)
	}
}

func TestHandleQueryInsert(t *testing.T) {
	mux := setup(t)
	mustPost(t, mux, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")

	rec, resp := post(t, mux, "INSERT INTO users VALUES (1, 'Alice')")
	if rec.Code != http.StatusOK {
		t.Fatalf("ステータス = %d, エラー = %q", rec.Code, resp.Error)
	}
	if resp.Error != "" {
		t.Errorf("エラー = %q, want 空", resp.Error)
	}

	// 挿入されたことをSELECTで確認する。
	// affected_rows は executor が Result.AffectedRows を一度もセットしていないため
	// 常に0になる。既知の問題（project_issues.md）。
	_, resp = post(t, mux, "SELECT * FROM users")
	if len(resp.Rows) != 1 {
		t.Fatalf("レコード数 = %d, want 1", len(resp.Rows))
	}
}

func TestHandleQuerySelect(t *testing.T) {
	mux := setup(t)
	mustPost(t, mux, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	mustPost(t, mux, "INSERT INTO users VALUES (1, 'Alice')")
	mustPost(t, mux, "INSERT INTO users VALUES (2, 'Bob')")

	rec, resp := post(t, mux, "SELECT * FROM users")
	if rec.Code != http.StatusOK {
		t.Fatalf("ステータス = %d, エラー = %q", rec.Code, resp.Error)
	}
	if len(resp.Columns) != 2 {
		t.Fatalf("カラム数 = %d, want 2", len(resp.Columns))
	}
	if resp.Columns[0] != "id" || resp.Columns[1] != "name" {
		t.Errorf("columns = %v, want [id name]", resp.Columns)
	}
	if len(resp.Rows) != 2 {
		t.Fatalf("レコード数 = %d, want 2", len(resp.Rows))
	}
	// JSONの数値はfloat64にデコードされる
	if resp.Rows[0][0] != float64(1) {
		t.Errorf("1件目のid = %v, want 1", resp.Rows[0][0])
	}
	if resp.Rows[0][1] != "Alice" {
		t.Errorf("1件目のname = %v, want Alice", resp.Rows[0][1])
	}
}

func TestHandleQuerySelectWhere(t *testing.T) {
	mux := setup(t)
	mustPost(t, mux, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	mustPost(t, mux, "INSERT INTO users VALUES (1, 'Alice')")
	mustPost(t, mux, "INSERT INTO users VALUES (2, 'Bob')")

	rec, resp := post(t, mux, "SELECT * FROM users WHERE id = 2")
	if rec.Code != http.StatusOK {
		t.Fatalf("ステータス = %d, エラー = %q", rec.Code, resp.Error)
	}
	if len(resp.Rows) != 1 {
		t.Fatalf("レコード数 = %d, want 1", len(resp.Rows))
	}
	if resp.Rows[0][1] != "Bob" {
		t.Errorf("name = %v, want Bob", resp.Rows[0][1])
	}
}

func TestHandleQueryDelete(t *testing.T) {
	mux := setup(t)
	mustPost(t, mux, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	mustPost(t, mux, "INSERT INTO users VALUES (1, 'Alice')")
	mustPost(t, mux, "INSERT INTO users VALUES (2, 'Bob')")

	rec, resp := post(t, mux, "DELETE FROM users WHERE id = 1")
	if rec.Code != http.StatusOK {
		t.Fatalf("ステータス = %d, エラー = %q", rec.Code, resp.Error)
	}

	_, resp = post(t, mux, "SELECT * FROM users")
	if len(resp.Rows) != 1 {
		t.Fatalf("DELETE後のレコード数 = %d, want 1", len(resp.Rows))
	}
	if resp.Rows[0][1] != "Bob" {
		t.Errorf("残ったname = %v, want Bob", resp.Rows[0][1])
	}
}

func TestHandleQueryUpdate(t *testing.T) {
	mux := setup(t)
	mustPost(t, mux, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	mustPost(t, mux, "INSERT INTO users VALUES (1, 'Alice')")
	mustPost(t, mux, "INSERT INTO users VALUES (2, 'Bob')")

	rec, resp := post(t, mux, "UPDATE users SET name = 'Carol' WHERE id = 1")
	if rec.Code != http.StatusOK {
		t.Fatalf("ステータス = %d, エラー = %q", rec.Code, resp.Error)
	}

	_, resp = post(t, mux, "SELECT * FROM users WHERE id = 1")
	if len(resp.Rows) != 1 {
		t.Fatalf("UPDATE後のレコード数 = %d, want 1", len(resp.Rows))
	}
	if resp.Rows[0][1] != "Carol" {
		t.Errorf("name = %v, want Carol", resp.Rows[0][1])
	}
}

// PKを降順にINSERTしても、全件がIndexScanで引けることを確認する。
// 葉ノードのスロット配列がキー順に保たれていないと取りこぼす。
func TestHandleQueryInsertDescendingPK(t *testing.T) {
	mux := setup(t)
	mustPost(t, mux, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	mustPost(t, mux, "INSERT INTO users VALUES (3, 'Carol')")
	mustPost(t, mux, "INSERT INTO users VALUES (2, 'Bob')")
	mustPost(t, mux, "INSERT INTO users VALUES (1, 'Alice')")

	for _, id := range []string{"1", "2", "3"} {
		_, resp := post(t, mux, "SELECT * FROM users WHERE id = "+id)
		if len(resp.Rows) != 1 {
			t.Errorf("id = %s のレコード数 = %d, want 1", id, len(resp.Rows))
		}
	}

	// Scanの結果もPK昇順で返る必要がある
	_, resp := post(t, mux, "SELECT * FROM users")
	if len(resp.Rows) != 3 {
		t.Fatalf("レコード数 = %d, want 3", len(resp.Rows))
	}
	for i, want := range []float64{1, 2, 3} {
		if resp.Rows[i][0] != want {
			t.Errorf("%d件目のid = %v, want %v", i+1, resp.Rows[i][0], want)
		}
	}
}

func TestHandleQueryEmptyResult(t *testing.T) {
	mux := setup(t)
	mustPost(t, mux, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")

	rec, resp := post(t, mux, "SELECT * FROM users")
	if rec.Code != http.StatusOK {
		t.Fatalf("ステータス = %d, エラー = %q", rec.Code, resp.Error)
	}
	// 結果が0件のときはomitemptyによりcolumns・rowsが省略される
	if len(resp.Rows) != 0 {
		t.Errorf("レコード数 = %d, want 0", len(resp.Rows))
	}
	if resp.Error != "" {
		t.Errorf("エラー = %q, want 空", resp.Error)
	}
}

func TestHandleQueryNullValue(t *testing.T) {
	mux := setup(t)
	mustPost(t, mux, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	mustPost(t, mux, "INSERT INTO users VALUES (1, NULL)")

	rec, resp := post(t, mux, "SELECT * FROM users")
	if rec.Code != http.StatusOK {
		t.Fatalf("ステータス = %d, エラー = %q", rec.Code, resp.Error)
	}
	if len(resp.Rows) != 1 {
		t.Fatalf("レコード数 = %d, want 1", len(resp.Rows))
	}
	if resp.Rows[0][1] != nil {
		t.Errorf("name = %v, want nil", resp.Rows[0][1])
	}
}

func TestHandleQueryContentType(t *testing.T) {
	mux := setup(t)

	rec, _ := post(t, mux, "CREATE TABLE users (id INT PRIMARY KEY)")
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// ---- 異常系 ----

func TestHandleQueryInvalidJSON(t *testing.T) {
	mux := setup(t)

	rec := postRaw(t, mux, "{invalid json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ステータス = %d, want 400", rec.Code)
	}
}

func TestHandleQueryEmptySQL(t *testing.T) {
	mux := setup(t)

	rec, resp := post(t, mux, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ステータス = %d, want 400", rec.Code)
	}
	if resp.Error == "" {
		t.Error("エラーメッセージが空")
	}
}

func TestHandleQueryParseError(t *testing.T) {
	mux := setup(t)

	rec, resp := post(t, mux, "SELECT FROM")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ステータス = %d, want 400", rec.Code)
	}
	if resp.Error == "" {
		t.Error("エラーメッセージが空")
	}
}

func TestHandleQueryPlanError(t *testing.T) {
	mux := setup(t)

	// テーブルが存在しないのでプランナーでエラーになる
	rec, resp := post(t, mux, "SELECT * FROM nonexistent")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ステータス = %d, want 400", rec.Code)
	}
	if resp.Error == "" {
		t.Error("エラーメッセージが空")
	}
}

func TestHandleQueryDropTableNotFound(t *testing.T) {
	mux := setup(t)

	rec, resp := post(t, mux, "DROP TABLE nonexistent")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ステータス = %d, want 400", rec.Code)
	}
	if resp.Error == "" {
		t.Error("エラーメッセージが空")
	}
}
