package planner

import (
	"fmt"

	"github.com/Moku3956/Project-D/sql/ast"
	"github.com/Moku3956/Project-D/types"
)

// PlanNode

type PlanNode interface {
	planNode()
	Kind() string
}

// テーブルを全件スキャンする
type SequentialScanNode struct {
	Table  string
	Schema *types.Schema
}

// B+TreeのPKで絞り込んでスキャンする
type IndexScanNode struct {
	Table  string
	Schema *types.Schema
	PkExpr ast.Expression // WHERE句のPK比較式
}

// WHERE条件で行を絞り込む
type FilterNode struct {
	Child     PlanNode
	Condition ast.Expression
}

// SELECT句で指定されたカラムだけを残す
type ProjectionNode struct {
	Child   PlanNode
	Columns []ast.Expression
}

// 左右のテーブルを全件突き合わせてJOINする
type NestedLoopJoinNode struct {
	Left      PlanNode
	Right     PlanNode
	Condition ast.Expression // ON条件
}

// ORDER BYで並び替える
type SortNode struct {
	Child  PlanNode
	Column string
	Desc   bool
}

// LIMITで件数を絞る
type LimitNode struct {
	Child PlanNode
	Count int
}

// 行を1件挿入する
type InsertNode struct {
	Table  string
	Schema *types.Schema
	Values []ast.Expression
}

// 条件に一致する行を更新する
type UpdateNode struct {
	Table       string
	Schema      *types.Schema
	Assignments []ast.Assignment
	Where       ast.Expression
}

// 条件に一致する行を削除する
type DeleteNode struct {
	Table  string
	Schema *types.Schema
	Where  ast.Expression
}

// テーブルを作成する
type CreateTableNode struct {
	Stmt *ast.CreateTableStatement
}

// テーブルを削除する
type DropTableNode struct {
	TableName string
}

type BeginNode struct{}
type CommitNode struct{}
type RollbackNode struct{}

func (n *SequentialScanNode) planNode()    {}
func (n *IndexScanNode) planNode()         {}
func (n *FilterNode) planNode()            {}
func (n *ProjectionNode) planNode()        {}
func (n *NestedLoopJoinNode) planNode()    {}
func (n *SortNode) planNode()              {}
func (n *LimitNode) planNode()             {}
func (n *InsertNode) planNode()            {}
func (n *UpdateNode) planNode()            {}
func (n *DeleteNode) planNode()            {}
func (n *CreateTableNode) planNode()       {}
func (n *DropTableNode) planNode()         {}
func (n *BeginNode) planNode()             {}
func (n *CommitNode) planNode()            {}
func (n *RollbackNode) planNode()          {}

func (n *SequentialScanNode) Kind() string { return "SequentialScan" }
func (n *IndexScanNode) Kind() string      { return "IndexScan" }
func (n *FilterNode) Kind() string         { return "Filter" }
func (n *ProjectionNode) Kind() string     { return "Projection" }
func (n *NestedLoopJoinNode) Kind() string { return "NestedLoopJoin" }
func (n *SortNode) Kind() string           { return "Sort" }
func (n *LimitNode) Kind() string          { return "Limit" }
func (n *InsertNode) Kind() string         { return "Insert" }
func (n *UpdateNode) Kind() string         { return "Update" }
func (n *DeleteNode) Kind() string         { return "Delete" }
func (n *CreateTableNode) Kind() string    { return "CreateTable" }
func (n *DropTableNode) Kind() string      { return "DropTable" }
func (n *BeginNode) Kind() string          { return "Begin" }
func (n *CommitNode) Kind() string         { return "Commit" }
func (n *RollbackNode) Kind() string       { return "Rollback" }

// catalogReader はプランナーが必要とするカタログ操作。
type catalogReader interface {
	GetSchema(table string) (*types.Schema, error)
	TableExists(table string) bool
}

type Planner struct {
	catalog catalogReader
}

func NewPlanner(catalog catalogReader) *Planner {
	return &Planner{catalog: catalog}
}

type PlanError struct {
	Message string
}

func (e *PlanError) Error() string { return e.Message }

func (p *Planner) Plan(stmt ast.Statement) (PlanNode, error) {
	switch s := stmt.(type) {
	case *ast.SelectStatement:
		return p.planSelect(s)
	case *ast.InsertStatement:
		return p.planInsert(s)
	case *ast.UpdateStatement:
		return p.planUpdate(s)
	case *ast.DeleteStatement:
		return p.planDelete(s)
	case *ast.CreateTableStatement:
		return &CreateTableNode{Stmt: s}, nil
	case *ast.DropTableStatement:
		if !p.catalog.TableExists(s.TableName) {
			return nil, &PlanError{Message: fmt.Sprintf("テーブル %q が存在しません", s.TableName)}
		}
		return &DropTableNode{TableName: s.TableName}, nil
	case *ast.BeginStatement:
		return &BeginNode{}, nil
	case *ast.CommitStatement:
		return &CommitNode{}, nil
	case *ast.RollbackStatement:
		return &RollbackNode{}, nil
	default:
		return nil, &PlanError{Message: "未対応のSQL文です"}
	}
}

func (p *Planner) planSelect(s *ast.SelectStatement) (PlanNode, error) {
	schema, err := p.catalog.GetSchema(s.Table)
	if err != nil {
		return nil, &PlanError{Message: fmt.Sprintf("テーブル %q が存在しません", s.Table)}
	}

	var scan PlanNode
	if s.Join != nil {
		rightSchema, err := p.catalog.GetSchema(s.Join.Table)
		if err != nil {
			return nil, &PlanError{Message: fmt.Sprintf("テーブル %q が存在しません", s.Join.Table)}
		}
		// 無条件で全件スキャンしている、改善余地あり
		left := &SequentialScanNode{Table: s.Table, Schema: schema}
		right := &SequentialScanNode{Table: s.Join.Table, Schema: rightSchema}
		scan = &NestedLoopJoinNode{Left: left, Right: right, Condition: s.Join.Condition}
	} else {
		scan = p.chooseScan(s.Table, schema, s.Where)
	}

	var node PlanNode = scan

	if s.Where != nil {
		if _, ok := scan.(*IndexScanNode); !ok {
			node = &FilterNode{Child: node, Condition: s.Where}
		}
	}

	node = &ProjectionNode{Child: node, Columns: s.Columns}

	if s.OrderBy != nil {
		node = &SortNode{Child: node, Column: s.OrderBy.Column, Desc: s.OrderBy.Desc}
	}

	if s.Limit != nil {
		node = &LimitNode{Child: node, Count: *s.Limit}
	}

	return node, nil
}

// chooseScan はWHERE条件を見てIndexScanかSequentialScanを選ぶ。
func (p *Planner) chooseScan(table string, schema *types.Schema, where ast.Expression) PlanNode {
	if where != nil {
		pkIdx := schema.PrimaryKeyIndex()
		if pkIdx >= 0 {
			pkName := schema.Columns[pkIdx].Name
			if pkExpr, ok := extractPKExpr(where, pkName); ok {
				return &IndexScanNode{Table: table, Schema: schema, PkExpr: pkExpr}
			}
		}
	}
	return &SequentialScanNode{Table: table, Schema: schema}
}

// extractPKExpr はWHERE条件がPKへの比較式かどうかを判定する。
func extractPKExpr(expr ast.Expression, pkName string) (ast.Expression, bool) {
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return nil, false
	}
	switch bin.Operator {
	case ast.OpEQ, ast.OpLT, ast.OpGT, ast.OpLTE, ast.OpGTE:
	default:
		return nil, false
	}
	ident, ok := bin.Left.(*ast.Identifier)
	if !ok || ident.Column != pkName {
		return nil, false
	}
	return expr, true
}

func (p *Planner) planInsert(s *ast.InsertStatement) (PlanNode, error) {
	schema, err := p.catalog.GetSchema(s.Table)
	if err != nil {
		return nil, &PlanError{Message: fmt.Sprintf("テーブル %q が存在しません", s.Table)}
	}
	if len(s.Values) != len(schema.Columns) {
		return nil, &PlanError{Message: fmt.Sprintf("カラム数が一致しません: 期待 %d, 実際 %d", len(schema.Columns), len(s.Values))}
	}
	return &InsertNode{Table: s.Table, Schema: schema, Values: s.Values}, nil
}

func (p *Planner) planUpdate(s *ast.UpdateStatement) (PlanNode, error) {
	schema, err := p.catalog.GetSchema(s.Table)
	if err != nil {
		return nil, &PlanError{Message: fmt.Sprintf("テーブル %q が存在しません", s.Table)}
	}
	return &UpdateNode{Table: s.Table, Schema: schema, Assignments: s.Assignments, Where: s.Where}, nil
}

func (p *Planner) planDelete(s *ast.DeleteStatement) (PlanNode, error) {
	schema, err := p.catalog.GetSchema(s.Table)
	if err != nil {
		return nil, &PlanError{Message: fmt.Sprintf("テーブル %q が存在しません", s.Table)}
	}
	return &DeleteNode{Table: s.Table, Schema: schema, Where: s.Where}, nil
}
