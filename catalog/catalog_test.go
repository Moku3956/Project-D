package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Moku3956/Project-D/types"
)

// ---- ヘルパー ----

// tmpPath はテスト用の一時ファイルパスを返す。テスト終了時に自動削除される。
func tmpPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "catalog.json")
}

// usersSchema はテスト用のスキーマを返す。
func usersSchema() types.Schema {
	return types.Schema{
		TableName: "users",
		Columns: []types.Column{
			{Name: "id", Type: types.DataType{Kind: types.KindIntType}, PrimaryKey: true, NotNull: true},
			{Name: "name", Type: types.DataType{Kind: types.KindVarcharType, Length: 50}},
		},
	}
}

// mustCreateTable はテーブルを作成する。エラーがあればt.Fatalする。
func mustCreateTable(t *testing.T, c *Catalog, schema types.Schema) {
	t.Helper()
	if err := c.CreateTable(schema); err != nil {
		t.Fatalf("CreateTable error: %v", err)
	}
}

// ---- 正常系 ----

func TestNewCatalogEmpty(t *testing.T) {
	c, err := NewCatalog(tmpPath(t))
	if err != nil {
		t.Fatalf("NewCatalog error: %v", err)
	}
	if len(c.schemas) != 0 {
		t.Errorf("schemasの件数 = %d, want 0", len(c.schemas))
	}
}

func TestCreateTable(t *testing.T) {
	c, _ := NewCatalog(tmpPath(t))
	if err := c.CreateTable(usersSchema()); err != nil {
		t.Fatalf("CreateTable error: %v", err)
	}
	if !c.TableExists("users") {
		t.Error("usersが登録されていない")
	}
}

func TestGetSchema(t *testing.T) {
	c, _ := NewCatalog(tmpPath(t))
	mustCreateTable(t, c, usersSchema())

	s, err := c.GetSchema("users")
	if err != nil {
		t.Fatalf("GetSchema error: %v", err)
	}
	if s.TableName != "users" {
		t.Errorf("TableName = %q, want %q", s.TableName, "users")
	}
	if len(s.Columns) != 2 {
		t.Errorf("カラム数 = %d, want 2", len(s.Columns))
	}
}

func TestTableExists(t *testing.T) {
	c, _ := NewCatalog(tmpPath(t))
	mustCreateTable(t, c, usersSchema())

	if !c.TableExists("users") {
		t.Error("usersが存在するはずなのにfalse")
	}
	if c.TableExists("orders") {
		t.Error("ordersは存在しないはずなのにtrue")
	}
}

func TestDropTable(t *testing.T) {
	c, _ := NewCatalog(tmpPath(t))
	mustCreateTable(t, c, usersSchema())

	if err := c.DropTable("users"); err != nil {
		t.Fatalf("DropTable error: %v", err)
	}
	if c.TableExists("users") {
		t.Error("usersが削除されていない")
	}
}

func TestTableIDIncrement(t *testing.T) {
	c, _ := NewCatalog(tmpPath(t))

	mustCreateTable(t, c, usersSchema())
	s1, _ := c.GetSchema("users")

	orders := types.Schema{
		TableName: "orders",
		Columns: []types.Column{
			{Name: "id", Type: types.DataType{Kind: types.KindIntType}, PrimaryKey: true},
		},
	}
	mustCreateTable(t, c, orders)
	s2, _ := c.GetSchema("orders")

	if s1.TableID == s2.TableID {
		t.Errorf("TableIDが重複している: %d", s1.TableID)
	}
}

// ---- 永続化 ----

func TestSaveAndLoad(t *testing.T) {
	path := tmpPath(t)

	c1, _ := NewCatalog(path)
	mustCreateTable(t, c1, usersSchema())

	// 別のCatalogインスタンスで同じファイルを読み込む
	c2, err := NewCatalog(path)
	if err != nil {
		t.Fatalf("NewCatalog (load) error: %v", err)
	}
	if !c2.TableExists("users") {
		t.Error("再起動後にusersが復元されていない")
	}
	s, _ := c2.GetSchema("users")
	if len(s.Columns) != 2 {
		t.Errorf("カラム数 = %d, want 2", len(s.Columns))
	}
}

func TestDropTablePersisted(t *testing.T) {
	path := tmpPath(t)

	c1, _ := NewCatalog(path)
	mustCreateTable(t, c1, usersSchema())
	if err := c1.DropTable("users"); err != nil {
		t.Fatalf("DropTable error: %v", err)
	}

	c2, err := NewCatalog(path)
	if err != nil {
		t.Fatalf("NewCatalog (load) error: %v", err)
	}
	if c2.TableExists("users") {
		t.Error("DropTable後に再起動してもusersが残っている")
	}
}

func TestNextTableIDPersisted(t *testing.T) {
	path := tmpPath(t)

	c1, _ := NewCatalog(path)
	mustCreateTable(t, c1, usersSchema())
	idBefore := c1.nextTableID

	c2, _ := NewCatalog(path)
	if c2.nextTableID != idBefore {
		t.Errorf("nextTableID = %d, want %d", c2.nextTableID, idBefore)
	}
}

// ---- 異常系 ----

func TestCreateTableDuplicate(t *testing.T) {
	c, _ := NewCatalog(tmpPath(t))
	mustCreateTable(t, c, usersSchema())

	err := c.CreateTable(usersSchema())
	if err == nil {
		t.Fatal("重複テーブルでエラーが期待されたがnil")
	}
}

func TestGetSchemaNotFound(t *testing.T) {
	c, _ := NewCatalog(tmpPath(t))

	_, err := c.GetSchema("users")
	if err == nil {
		t.Fatal("存在しないテーブルでエラーが期待されたがnil")
	}
}

func TestDropTableNotFound(t *testing.T) {
	c, _ := NewCatalog(tmpPath(t))

	err := c.DropTable("users")
	if err == nil {
		t.Fatal("存在しないテーブルでエラーが期待されたがnil")
	}
}

func TestNewCatalogInvalidJSON(t *testing.T) {
	path := tmpPath(t)
	if err := os.WriteFile(path, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	_, err := NewCatalog(path)
	if err == nil {
		t.Fatal("不正なJSONでエラーが期待されたがnil")
	}
}
