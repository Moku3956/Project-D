package ast

import "github.com/Moku3956/Project-D/types"

// Statement

type Statement interface {
	statementNode()
	Kind() string
}

// 意図的に汎用性をなくしている(実装複雑化を回避)、将来的には汎用性を持たせる
type SelectStatement struct {
	Columns []Expression
	Table   string
	Join    *JoinClause
	Where   Expression
	OrderBy *OrderByClause
	Limit   *int
}

type InsertStatement struct {
	Table  string
	Values []Expression
}

type UpdateStatement struct {
	Table       string
	Assignments []Assignment
	Where       Expression
}

type Assignment struct {
	Column string
	Value  Expression
}

type DeleteStatement struct {
	Table string
	Where Expression
}

type CreateTableStatement struct {
	TableName string
	Columns   []types.Column
}

type DropTableStatement struct {
	TableName string
}

type BeginStatement struct{}
type CommitStatement struct{}
type RollbackStatement struct{}

func (s *SelectStatement) statementNode()      {}
func (s *InsertStatement) statementNode()      {}
func (s *UpdateStatement) statementNode()      {}
func (s *DeleteStatement) statementNode()      {}
func (s *CreateTableStatement) statementNode() {}
func (s *DropTableStatement) statementNode()   {}
func (s *BeginStatement) statementNode()       {}
func (s *CommitStatement) statementNode()      {}
func (s *RollbackStatement) statementNode()    {}

func (s *SelectStatement) Kind() string      { return "SELECT" }
func (s *InsertStatement) Kind() string      { return "INSERT" }
func (s *UpdateStatement) Kind() string      { return "UPDATE" }
func (s *DeleteStatement) Kind() string      { return "DELETE" }
func (s *CreateTableStatement) Kind() string { return "CREATE_TABLE" }
func (s *DropTableStatement) Kind() string   { return "DROP_TABLE" }
func (s *BeginStatement) Kind() string       { return "BEGIN" }
func (s *CommitStatement) Kind() string      { return "COMMIT" }
func (s *RollbackStatement) Kind() string    { return "ROLLBACK" }

type JoinClause struct {
	Table     string
	Condition Expression
}

type OrderByClause struct {
	Column string
	Desc   bool
}

// Expression

type Expression interface {
	expressionNode()
}

type Identifier struct {
	Table  string // テーブル修飾子（users.id の場合 "users"）
	Column string
}

type Wildcard struct{}

type IntLiteral struct {
	Value int64
}

type StringLiteral struct {
	Value string
}

type BoolLiteral struct {
	Value bool
}

type NullLiteral struct{}

type OperatorType int

const (
	OpEQ  OperatorType = iota
	OpNEQ
	OpLT
	OpGT
	OpLTE
	OpGTE
	OpAND
	OpOR
)

type BinaryExpr struct {
	Left     Expression
	Operator OperatorType
	Right    Expression
}

type UnaryExpr struct {
	Operator string
	Operand  Expression
}

type IsNullExpr struct {
	Operand Expression
}

type IsNotNullExpr struct {
	Operand Expression
}

func (e *Identifier) expressionNode()    {}
func (e *Wildcard) expressionNode()      {}
func (e *IntLiteral) expressionNode()    {}
func (e *StringLiteral) expressionNode() {}
func (e *BoolLiteral) expressionNode()   {}
func (e *NullLiteral) expressionNode()   {}
func (e *BinaryExpr) expressionNode()    {}
func (e *UnaryExpr) expressionNode()     {}
func (e *IsNullExpr) expressionNode()    {}
func (e *IsNotNullExpr) expressionNode() {}
