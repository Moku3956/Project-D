package parser

import (
	"testing"

	"github.com/Moku3956/Project-D/sql/ast"
	"github.com/Moku3956/Project-D/types"
)

// toParseErrors はerrorからParseErrorsを取り出す。
func toParseErrors(err error) ParseErrors {
	pe, _ := err.(ParseErrors)
	return pe
}

// hasKind はerrorの中に指定したKindのエラーが含まれるかを返す。
func hasKind(err error, kind ParseErrorKind) bool {
	for _, e := range toParseErrors(err) {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

// ---- 正常系 ----

func TestParseSelect(t *testing.T) {
	stmt, err := Parse("SELECT id, name FROM users")
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	sel, ok := stmt.(*ast.SelectStatement)
	if !ok {
		t.Fatal("SelectStatement でない")
	}
	if sel.Table != "users" {
		t.Errorf("Table = %q, want %q", sel.Table, "users")
	}
	if len(sel.Columns) != 2 {
		t.Fatalf("Columns len = %d, want 2", len(sel.Columns))
	}
	id, ok := sel.Columns[0].(*ast.Identifier)
	if !ok || id.Column != "id" {
		t.Errorf("Columns[0] = %v, want Identifier{Column:\"id\"}", sel.Columns[0])
	}
	name, ok := sel.Columns[1].(*ast.Identifier)
	if !ok || name.Column != "name" {
		t.Errorf("Columns[1] = %v, want Identifier{Column:\"name\"}", sel.Columns[1])
	}
}

func TestParseSelectWhere(t *testing.T) {
	stmt, err := Parse("SELECT id FROM users WHERE age > 20 AND active = true")
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	sel := stmt.(*ast.SelectStatement)
	if sel.Where == nil {
		t.Fatal("Where が nil")
	}
	bin, ok := sel.Where.(*ast.BinaryExpr)
	if !ok || bin.Operator != ast.OpAND {
		t.Errorf("Where の root が AND でない: %T", sel.Where)
	}
}

func TestParseSelectOrderByLimit(t *testing.T) {
	stmt, err := Parse("SELECT id FROM users ORDER BY age DESC LIMIT 10")
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	sel := stmt.(*ast.SelectStatement)
	if sel.OrderBy == nil {
		t.Fatal("OrderBy が nil")
	}
	if sel.OrderBy.Column != "age" || !sel.OrderBy.Desc {
		t.Errorf("OrderBy = %+v, want {Column:age Desc:true}", sel.OrderBy)
	}
	if sel.Limit == nil || *sel.Limit != 10 {
		t.Errorf("Limit = %v, want 10", sel.Limit)
	}
}

func TestParseSelectJoin(t *testing.T) {
	stmt, err := Parse("SELECT id FROM users JOIN orders ON users.id = orders.user_id")
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	sel := stmt.(*ast.SelectStatement)
	if sel.Join == nil {
		t.Fatal("Join が nil")
	}
	if sel.Join.Table != "orders" {
		t.Errorf("Join.Table = %q, want %q", sel.Join.Table, "orders")
	}
	if sel.Join.Condition == nil {
		t.Fatal("Join.Condition が nil")
	}
}

func TestParseSelectWildcard(t *testing.T) {
	stmt, err := Parse("SELECT * FROM products")
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	sel := stmt.(*ast.SelectStatement)
	if len(sel.Columns) != 1 {
		t.Fatalf("Columns len = %d, want 1", len(sel.Columns))
	}
	if _, ok := sel.Columns[0].(*ast.Wildcard); !ok {
		t.Errorf("Columns[0] が Wildcard でない: %T", sel.Columns[0])
	}
}

func TestParseInsert(t *testing.T) {
	stmt, err := Parse("INSERT INTO users VALUES (1, 'Alice')")
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	ins, ok := stmt.(*ast.InsertStatement)
	if !ok {
		t.Fatal("InsertStatement でない")
	}
	if ins.Table != "users" {
		t.Errorf("Table = %q, want %q", ins.Table, "users")
	}
	if len(ins.Values) != 2 {
		t.Fatalf("Values len = %d, want 2", len(ins.Values))
	}
	iv, ok := ins.Values[0].(*ast.IntLiteral)
	if !ok || iv.Value != 1 {
		t.Errorf("Values[0] = %v, want IntLiteral{1}", ins.Values[0])
	}
	sv, ok := ins.Values[1].(*ast.StringLiteral)
	if !ok || sv.Value != "Alice" {
		t.Errorf("Values[1] = %v, want StringLiteral{Alice}", ins.Values[1])
	}
}

func TestParseUpdate(t *testing.T) {
	stmt, err := Parse("UPDATE users SET name = 'Bob', age = 30 WHERE id = 1")
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	upd, ok := stmt.(*ast.UpdateStatement)
	if !ok {
		t.Fatal("UpdateStatement でない")
	}
	if upd.Table != "users" {
		t.Errorf("Table = %q, want %q", upd.Table, "users")
	}
	if len(upd.Assignments) != 2 {
		t.Fatalf("Assignments len = %d, want 2", len(upd.Assignments))
	}
	if upd.Assignments[0].Column != "name" {
		t.Errorf("Assignments[0].Column = %q, want %q", upd.Assignments[0].Column, "name")
	}
	if upd.Where == nil {
		t.Fatal("Where が nil")
	}
}

func TestParseDelete(t *testing.T) {
	stmt, err := Parse("DELETE FROM users WHERE id = 5")
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	del, ok := stmt.(*ast.DeleteStatement)
	if !ok {
		t.Fatal("DeleteStatement でない")
	}
	if del.Table != "users" {
		t.Errorf("Table = %q, want %q", del.Table, "users")
	}
	if del.Where == nil {
		t.Fatal("Where が nil")
	}
}

func TestParseCreateTable(t *testing.T) {
	sql := "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50) NOT NULL)"
	stmt, err := Parse(sql)
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	ct, ok := stmt.(*ast.CreateTableStatement)
	if !ok {
		t.Fatal("CreateTableStatement でない")
	}
	if ct.TableName != "users" {
		t.Errorf("TableName = %q, want %q", ct.TableName, "users")
	}
	if len(ct.Columns) != 2 {
		t.Fatalf("Columns len = %d, want 2", len(ct.Columns))
	}
	if !ct.Columns[0].PrimaryKey || ct.Columns[0].Type.Kind != types.KindIntType {
		t.Errorf("Columns[0] = %+v, want INT PRIMARY KEY", ct.Columns[0])
	}
	if ct.Columns[1].Type.Kind != types.KindVarcharType || ct.Columns[1].Type.Length != 50 {
		t.Errorf("Columns[1] = %+v, want VARCHAR(50)", ct.Columns[1])
	}
}

func TestParseDropTable(t *testing.T) {
	stmt, err := Parse("DROP TABLE users")
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	dt, ok := stmt.(*ast.DropTableStatement)
	if !ok {
		t.Fatal("DropTableStatement でない")
	}
	if dt.TableName != "users" {
		t.Errorf("TableName = %q, want %q", dt.TableName, "users")
	}
}

func TestParseBeginCommitRollback(t *testing.T) {
	cases := []struct {
		sql  string
		kind string
	}{
		{"BEGIN", "BEGIN"},
		{"COMMIT", "COMMIT"},
		{"ROLLBACK", "ROLLBACK"},
	}
	for _, c := range cases {
		stmt, err := Parse(c.sql)
		if err != nil {
			t.Errorf("%s: エラーが発生: %v", c.sql, err)
			continue
		}
		if stmt.Kind() != c.kind {
			t.Errorf("%s: Kind = %q, want %q", c.sql, stmt.Kind(), c.kind)
		}
	}
}

func TestParseIsNull(t *testing.T) {
	stmt, err := Parse("SELECT id FROM users WHERE name IS NULL")
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	sel := stmt.(*ast.SelectStatement)
	if _, ok := sel.Where.(*ast.IsNullExpr); !ok {
		t.Errorf("Where = %T, want IsNullExpr", sel.Where)
	}

	stmt2, err := Parse("SELECT id FROM users WHERE name IS NOT NULL")
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	sel2 := stmt2.(*ast.SelectStatement)
	if _, ok := sel2.Where.(*ast.IsNotNullExpr); !ok {
		t.Errorf("Where = %T, want IsNotNullExpr", sel2.Where)
	}
}

func TestParseNotExpr(t *testing.T) {
	stmt, err := Parse("SELECT id FROM users WHERE NOT active = true")
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	sel := stmt.(*ast.SelectStatement)
	unary, ok := sel.Where.(*ast.UnaryExpr)
	if !ok || unary.Operator != "NOT" {
		t.Errorf("Where = %T, want UnaryExpr{NOT}", sel.Where)
	}
}

// ---- 異常系 ----

func TestParseSelectMissingColumns(t *testing.T) {
	_, err := Parse("SELECT")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
	if !hasKind(err, MissingColumnList) {
		t.Errorf("MissingColumnList が含まれない: %v", err)
	}
}

func TestParseSelectMissingFrom(t *testing.T) {
	_, err := Parse("SELECT id name")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
	if !hasKind(err, MissingFromClause) {
		t.Errorf("MissingFromClause が含まれない: %v", err)
	}
}

func TestParseSelectMissingTable(t *testing.T) {
	_, err := Parse("SELECT id FROM")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
	if !hasKind(err, UnexpectedToken) {
		t.Errorf("UnexpectedToken が含まれない: %v", err)
	}
}

func TestParseInsertMissingValues(t *testing.T) {
	_, err := Parse("INSERT INTO users")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
	if !hasKind(err, MissingToken) {
		t.Errorf("MissingToken が含まれない: %v", err)
	}
}

func TestParseUpdateMissingSet(t *testing.T) {
	_, err := Parse("UPDATE users")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
	if !hasKind(err, MissingToken) {
		t.Errorf("MissingToken が含まれない: %v", err)
	}
}

func TestParseCreateTableNoColumns(t *testing.T) {
	_, err := Parse("CREATE TABLE t ()")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
	if !hasKind(err, MissingColumnDef) {
		t.Errorf("MissingColumnDef が含まれない: %v", err)
	}
}

func TestParseCreateTableInvalidType(t *testing.T) {
	_, err := Parse("CREATE TABLE t (id BLOOP)")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
	if !hasKind(err, InvalidDataType) {
		t.Errorf("InvalidDataType が含まれない: %v", err)
	}
}

func TestParseCreateTableDuplicatePK(t *testing.T) {
	sql := "CREATE TABLE t (id INT PRIMARY KEY, code INT PRIMARY KEY)"
	_, err := Parse(sql)
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
	if !hasKind(err, DuplicatePrimaryKey) {
		t.Errorf("DuplicatePrimaryKey が含まれない: %v", err)
	}
}

func TestParseLimitNotInt(t *testing.T) {
	_, err := Parse("SELECT id FROM users LIMIT abc")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
	if !hasKind(err, InvalidLiteral) {
		t.Errorf("InvalidLiteral が含まれない: %v", err)
	}
}

func TestParseJoinMissingTable(t *testing.T) {
	_, err := Parse("SELECT id FROM users JOIN WHERE id = 1")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
	if !hasKind(err, MissingJoinTable) {
		t.Errorf("MissingJoinTable が含まれない: %v", err)
	}
}

func TestParseJoinMissingOn(t *testing.T) {
	_, err := Parse("SELECT id FROM users JOIN orders WHERE id = 1")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
	if !hasKind(err, MissingJoinCondition) {
		t.Errorf("MissingJoinCondition が含まれない: %v", err)
	}
}

func TestParseUnexpectedToken(t *testing.T) {
	_, err := Parse("INVALID STATEMENT")
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
	if !hasKind(err, UnexpectedToken) {
		t.Errorf("UnexpectedToken が含まれない: %v", err)
	}
}
