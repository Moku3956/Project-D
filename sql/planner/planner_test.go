package planner

import (
	"fmt"
	"testing"

	"github.com/Moku3956/Project-D/sql/ast"
	"github.com/Moku3956/Project-D/types"
)

// mockCatalog はテスト用のcatalogReader実装。
type mockCatalog struct {
	schemas map[string]*types.Schema
}

func (m *mockCatalog) GetSchema(table string) (*types.Schema, error) {
	s, ok := m.schemas[table]
	if !ok {
		return nil, fmt.Errorf("table %q not found", table)
	}
	return s, nil
}

func (m *mockCatalog) TableExists(table string) bool {
	_, ok := m.schemas[table]
	return ok
}

// usersSchema はテスト用のusersテーブルスキーマ。
func usersSchema() *types.Schema {
	return &types.Schema{
		TableName: "users",
		Columns: []types.Column{
			{Name: "id", Type: types.DataType{Kind: types.KindIntType}, PrimaryKey: true},
			{Name: "name", Type: types.DataType{Kind: types.KindVarcharType, Length: 50}},
		},
	}
}

// ordersSchema はテスト用のordersテーブルスキーマ。
func ordersSchema() *types.Schema {
	return &types.Schema{
		TableName: "orders",
		Columns: []types.Column{
			{Name: "id", Type: types.DataType{Kind: types.KindIntType}, PrimaryKey: true},
			{Name: "user_id", Type: types.DataType{Kind: types.KindIntType}},
		},
	}
}

func newPlanner(schemas map[string]*types.Schema) *Planner {
	return NewPlanner(&mockCatalog{schemas: schemas})
}

// ---- 正常系 ----

func TestPlanSelectSequentialScan(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{"users": usersSchema()})
	stmt := &ast.SelectStatement{
		Columns: []ast.Expression{&ast.Wildcard{}},
		Table:   "users",
	}
	node, err := pl.Plan(stmt)
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	proj, ok := node.(*ProjectionNode)
	if !ok {
		t.Fatalf("root = %T, want ProjectionNode", node)
	}
	if _, ok := proj.Child.(*SequentialScanNode); !ok {
		t.Errorf("Child = %T, want SequentialScanNode", proj.Child)
	}
}

func TestPlanSelectIndexScan(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{"users": usersSchema()})
	// WHERE id = 1
	where := &ast.BinaryExpr{
		Left:     &ast.Identifier{Column: "id"},
		Operator: ast.OpEQ,
		Right:    &ast.IntLiteral{Value: 1},
	}
	stmt := &ast.SelectStatement{
		Columns: []ast.Expression{&ast.Wildcard{}},
		Table:   "users",
		Where:   where,
	}
	node, err := pl.Plan(stmt)
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	proj := node.(*ProjectionNode)
	if _, ok := proj.Child.(*IndexScanNode); !ok {
		t.Errorf("Child = %T, want IndexScanNode", proj.Child)
	}
}

func TestPlanSelectFilter(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{"users": usersSchema()})
	// WHERE name = 'Alice'（PKでないカラム）
	where := &ast.BinaryExpr{
		Left:     &ast.Identifier{Column: "name"},
		Operator: ast.OpEQ,
		Right:    &ast.StringLiteral{Value: "Alice"},
	}
	stmt := &ast.SelectStatement{
		Columns: []ast.Expression{&ast.Wildcard{}},
		Table:   "users",
		Where:   where,
	}
	node, err := pl.Plan(stmt)
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	proj := node.(*ProjectionNode)
	filter, ok := proj.Child.(*FilterNode)
	if !ok {
		t.Fatalf("Child = %T, want FilterNode", proj.Child)
	}
	if _, ok := filter.Child.(*SequentialScanNode); !ok {
		t.Errorf("FilterNode.Child = %T, want SequentialScanNode", filter.Child)
	}
}

func TestPlanSelectProjection(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{"users": usersSchema()})
	cols := []ast.Expression{
		&ast.Identifier{Column: "id"},
		&ast.Identifier{Column: "name"},
	}
	stmt := &ast.SelectStatement{
		Columns: cols,
		Table:   "users",
	}
	node, err := pl.Plan(stmt)
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	proj, ok := node.(*ProjectionNode)
	if !ok {
		t.Fatalf("root = %T, want ProjectionNode", node)
	}
	if len(proj.Columns) != 2 {
		t.Errorf("Columns len = %d, want 2", len(proj.Columns))
	}
}

func TestPlanSelectOrderByLimit(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{"users": usersSchema()})
	n := 5
	stmt := &ast.SelectStatement{
		Columns: []ast.Expression{&ast.Wildcard{}},
		Table:   "users",
		OrderBy: &ast.OrderByClause{Column: "name", Desc: true},
		Limit:   &n,
	}
	node, err := pl.Plan(stmt)
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	limit, ok := node.(*LimitNode)
	if !ok {
		t.Fatalf("root = %T, want LimitNode", node)
	}
	if limit.Count != 5 {
		t.Errorf("Count = %d, want 5", limit.Count)
	}
	sort, ok := limit.Child.(*SortNode)
	if !ok {
		t.Fatalf("LimitNode.Child = %T, want SortNode", limit.Child)
	}
	if sort.Column != "name" || !sort.Desc {
		t.Errorf("SortNode = %+v, want {Column:name Desc:true}", sort)
	}
}

func TestPlanSelectJoin(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{
		"users":  usersSchema(),
		"orders": ordersSchema(),
	})
	stmt := &ast.SelectStatement{
		Columns: []ast.Expression{&ast.Wildcard{}},
		Table:   "users",
		Join: &ast.JoinClause{
			Table: "orders",
			Condition: &ast.BinaryExpr{
				Left:     &ast.Identifier{Table: "users", Column: "id"},
				Operator: ast.OpEQ,
				Right:    &ast.Identifier{Table: "orders", Column: "user_id"},
			},
		},
	}
	node, err := pl.Plan(stmt)
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	proj := node.(*ProjectionNode)
	join, ok := proj.Child.(*NestedLoopJoinNode)
	if !ok {
		t.Fatalf("Child = %T, want NestedLoopJoinNode", proj.Child)
	}
	if _, ok := join.Left.(*SequentialScanNode); !ok {
		t.Errorf("Left = %T, want SequentialScanNode", join.Left)
	}
	if _, ok := join.Right.(*SequentialScanNode); !ok {
		t.Errorf("Right = %T, want SequentialScanNode", join.Right)
	}
}

func TestPlanInsert(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{"users": usersSchema()})
	stmt := &ast.InsertStatement{
		Table:  "users",
		Values: []ast.Expression{&ast.IntLiteral{Value: 1}, &ast.StringLiteral{Value: "Alice"}},
	}
	node, err := pl.Plan(stmt)
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	ins, ok := node.(*InsertNode)
	if !ok {
		t.Fatalf("node = %T, want InsertNode", node)
	}
	if ins.Table != "users" {
		t.Errorf("Table = %q, want %q", ins.Table, "users")
	}
}

func TestPlanUpdate(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{"users": usersSchema()})
	stmt := &ast.UpdateStatement{
		Table:       "users",
		Assignments: []ast.Assignment{{Column: "name", Value: &ast.StringLiteral{Value: "Bob"}}},
		Where: &ast.BinaryExpr{
			Left:     &ast.Identifier{Column: "id"},
			Operator: ast.OpEQ,
			Right:    &ast.IntLiteral{Value: 1},
		},
	}
	node, err := pl.Plan(stmt)
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	if _, ok := node.(*UpdateNode); !ok {
		t.Fatalf("node = %T, want UpdateNode", node)
	}
}

func TestPlanDelete(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{"users": usersSchema()})
	stmt := &ast.DeleteStatement{
		Table: "users",
		Where: &ast.BinaryExpr{
			Left:     &ast.Identifier{Column: "id"},
			Operator: ast.OpEQ,
			Right:    &ast.IntLiteral{Value: 1},
		},
	}
	node, err := pl.Plan(stmt)
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	if _, ok := node.(*DeleteNode); !ok {
		t.Fatalf("node = %T, want DeleteNode", node)
	}
}

func TestPlanCreateTable(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{})
	stmt := &ast.CreateTableStatement{
		TableName: "users",
		Columns: []types.Column{
			{Name: "id", Type: types.DataType{Kind: types.KindIntType}, PrimaryKey: true},
		},
	}
	node, err := pl.Plan(stmt)
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	if _, ok := node.(*CreateTableNode); !ok {
		t.Fatalf("node = %T, want CreateTableNode", node)
	}
}

func TestPlanDropTable(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{"users": usersSchema()})
	stmt := &ast.DropTableStatement{TableName: "users"}
	node, err := pl.Plan(stmt)
	if err != nil {
		t.Fatalf("エラーが発生: %v", err)
	}
	if _, ok := node.(*DropTableNode); !ok {
		t.Fatalf("node = %T, want DropTableNode", node)
	}
}

func TestPlanBeginCommitRollback(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{})
	cases := []struct {
		stmt ast.Statement
		kind string
	}{
		{&ast.BeginStatement{}, "Begin"},
		{&ast.CommitStatement{}, "Commit"},
		{&ast.RollbackStatement{}, "Rollback"},
	}
	for _, c := range cases {
		node, err := pl.Plan(c.stmt)
		if err != nil {
			t.Errorf("%s: エラーが発生: %v", c.kind, err)
			continue
		}
		if node.Kind() != c.kind {
			t.Errorf("Kind = %q, want %q", node.Kind(), c.kind)
		}
	}
}

// ---- 異常系 ----

func TestPlanSelectTableNotFound(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{})
	stmt := &ast.SelectStatement{
		Columns: []ast.Expression{&ast.Wildcard{}},
		Table:   "users",
	}
	_, err := pl.Plan(stmt)
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
}

func TestPlanInsertColumnCountMismatch(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{"users": usersSchema()})
	// usersは2カラムだが値を1つしか渡さない
	stmt := &ast.InsertStatement{
		Table:  "users",
		Values: []ast.Expression{&ast.IntLiteral{Value: 1}},
	}
	_, err := pl.Plan(stmt)
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
}

func TestPlanDropTableNotFound(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{})
	stmt := &ast.DropTableStatement{TableName: "users"}
	_, err := pl.Plan(stmt)
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
}

func TestPlanJoinTableNotFound(t *testing.T) {
	pl := newPlanner(map[string]*types.Schema{"users": usersSchema()})
	stmt := &ast.SelectStatement{
		Columns: []ast.Expression{&ast.Wildcard{}},
		Table:   "users",
		Join: &ast.JoinClause{
			Table:     "orders", // 存在しないテーブル
			Condition: &ast.BoolLiteral{Value: true},
		},
	}
	_, err := pl.Plan(stmt)
	if err == nil {
		t.Fatal("エラーが期待されたが nil")
	}
}
