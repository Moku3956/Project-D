# AST 仕様

## 概要

パーサーが生成するSQL文の構文木。プランナーはASTを受け取って実行計画を作成する。

---

## Statement

```go
type Statement interface {
    statementNode()
    Kind() string
}
```

| 種別 | SQL |
|------|-----|
| `SelectStatement` | `SELECT ...` |
| `InsertStatement` | `INSERT INTO ...` |
| `UpdateStatement` | `UPDATE ...` |
| `DeleteStatement` | `DELETE FROM ...` |
| `CreateTableStatement` | `CREATE TABLE ...` |
| `DropTableStatement` | `DROP TABLE ...` |
| `BeginStatement` | `BEGIN` |
| `CommitStatement` | `COMMIT` |
| `RollbackStatement` | `ROLLBACK` |

### SelectStatement

```go
type SelectStatement struct {
    Columns []Expression
    Table   string
    Join    *JoinClause
    Where   Expression
    OrderBy *OrderByClause
    Limit   *int
}
```

### JoinClause

```go
type JoinClause struct {
    Table     string
    Condition Expression  // ON の条件
}
```

### OrderByClause

```go
type OrderByClause struct {
    Column string
    Desc   bool  // true=DESC、false=ASC
}
```

### CreateTableStatement

```go
type CreateTableStatement struct {
    TableName string
    Columns   []types.Column  // types.Column をそのまま使う
}
```

---

## Expression

```go
type Expression interface {
    expressionNode()
}
```

| 種別 | 例 |
|------|----|
| `Identifier` | `age`、`users.id` |
| `Wildcard` | `*` |
| `IntLiteral` | `20` |
| `StringLiteral` | `'Alice'` |
| `BoolLiteral` | `true`、`false` |
| `NullLiteral` | `NULL` |
| `BinaryExpr` | `age > 20`、`AND`、`OR` |
| `UnaryExpr` | `NOT` |
| `IsNullExpr` | `name IS NULL` |
| `IsNotNullExpr` | `name IS NOT NULL` |

### BinaryExpr

```go
type OperatorType int

const (
    EQ  OperatorType = iota  // =
    NEQ                       // !=
    LT                        // <
    GT                        // >
    LTE                       // <=
    GTE                       // >=
    AND
    OR
)

type BinaryExpr struct {
    Left     Expression
    Operator OperatorType
    Right    Expression
}
```

### UnaryExpr

```go
type UnaryExpr struct {
    Operator string  // "NOT"
    Operand  Expression
}
```
