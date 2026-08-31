package executor

import (
	"fmt"
	"sort"

	"github.com/Moku3956/Project-D/sql/ast"
	"github.com/Moku3956/Project-D/sql/planner"
	"github.com/Moku3956/Project-D/txn"
	"github.com/Moku3956/Project-D/types"
)

// TableRepository はストレージへのアクセスを抽象化する。
type TableRepository interface {
	OpenTable(table string, schema *types.Schema) error
	FindByPK(table string, pk types.Value) (*types.Row, error)
	Scan(table string) ([]types.Row, error)
	Insert(table string, row types.Row, txnID uint64) error
	Update(table string, pk types.Value, row types.Row, txnID uint64) error
	Delete(table string, pk types.Value, txnID uint64) error
}

// catalogReader はexecutorが必要とするカタログ操作。
type catalogReader interface {
	GetSchema(table string) (*types.Schema, error)
	TableExists(table string) bool
	CreateTable(schema types.Schema) error
	DropTable(table string) error
}

// Executor はVolcanoモデルの反復子。
type Executor interface {
	Next() (*types.Row, error)
	Schema() *types.Schema
	Close() error
}

// Result はSQL実行の結果。
type Result struct {
	Rows         []types.Row
	AffectedRows int
	Schema       *types.Schema
}

// Engine はプランノードをExecutorに変換して実行する。
type Engine struct {
	repo    TableRepository
	catalog catalogReader
	txnMgr  *txn.Manager
}

// NewEngine はEngineを生成する。
func NewEngine(repo TableRepository, catalog catalogReader, txnMgr *txn.Manager) *Engine {
	return &Engine{repo: repo, catalog: catalog, txnMgr: txnMgr}
}

// Execute はSQLを1文=1トランザクションの自動コミットで実行する。
func (e *Engine) Execute(node planner.PlanNode) (*Result, error) {
	if result, handled, err := e.executeDDLOrTxnMarker(node); handled {
		return result, err
	}

	t := e.txnMgr.Begin()
	result, err := e.execLockedInTxn(t, node)
	if err != nil {
		_ = e.txnMgr.Rollback(t)
		return nil, err
	}
	if err := e.txnMgr.Commit(t); err != nil {
		return nil, err
	}
	return result, nil
}

// ExecuteInTxn はtの中でSQLを実行する。コミット/ロールバックは呼び出し元の責任。
func (e *Engine) ExecuteInTxn(t *txn.Txn, node planner.PlanNode) (*Result, error) {
	if result, handled, err := e.executeDDLOrTxnMarker(node); handled {
		return result, err
	}
	return e.execLockedInTxn(t, node)
}

// executeDDLOrTxnMarker はトランザクション不要なノード(DDL・BEGIN/COMMIT/ROLLBACK)を処理する。
func (e *Engine) executeDDLOrTxnMarker(node planner.PlanNode) (result *Result, handled bool, err error) {
	switch n := node.(type) {
	case *planner.CreateTableNode:
		schema := types.Schema{
			TableName: n.Stmt.TableName,
			Columns:   n.Stmt.Columns,
		}
		if err := e.catalog.CreateTable(schema); err != nil {
			return nil, true, err
		}
		// TableID が付与されたスキーマをカタログから取得して登録する
		assigned, err := e.catalog.GetSchema(schema.TableName)
		if err != nil {
			return nil, true, err
		}
		if err := e.repo.OpenTable(schema.TableName, assigned); err != nil {
			return nil, true, err
		}
		return &Result{}, true, nil

	case *planner.DropTableNode:
		if err := e.catalog.DropTable(n.TableName); err != nil {
			return nil, true, err
		}
		return &Result{}, true, nil

	case *planner.BeginNode, *planner.CommitNode, *planner.RollbackNode:
		return &Result{}, true, nil
	}
	return nil, false, nil
}

// execLockedInTxn はtでロックを取得してnodeを実行する。
func (e *Engine) execLockedInTxn(t *txn.Txn, node planner.PlanNode) (*Result, error) {
	if err := e.lockFor(t, node); err != nil {
		return nil, err
	}
	return e.executeLocked(node, t.ID)
}

// lockFor はノードが読み書きするテーブルのロックを取得する。(読み込みと書き込みクエリのロックを分ける)
func (e *Engine) lockFor(t *txn.Txn, node planner.PlanNode) error {
	switch n := node.(type) {
	case *planner.InsertNode:
		return e.txnMgr.Lock(t, n.Table)
	case *planner.UpdateNode:
		return e.txnMgr.Lock(t, n.Table)
	case *planner.DeleteNode:
		return e.txnMgr.Lock(t, n.Table)
	default:
		for _, table := range tableNamesOf(node) {
			if err := e.txnMgr.RLock(t, table); err != nil {
				return err
			}
		}
		return nil
	}
}

// tableNamesOf はプランノードの木を辿り、スキャン対象のテーブル名を集める。
func tableNamesOf(node planner.PlanNode) []string {
	switch n := node.(type) {
	case *planner.SequentialScanNode:
		return []string{n.Table}
	case *planner.IndexScanNode:
		return []string{n.Table}
	case *planner.FilterNode:
		return tableNamesOf(n.Child)
	case *planner.ProjectionNode:
		return tableNamesOf(n.Child)
	case *planner.NestedLoopJoinNode:
		return append(tableNamesOf(n.Left), tableNamesOf(n.Right)...)
	case *planner.SortNode:
		return tableNamesOf(n.Child)
	case *planner.LimitNode:
		return tableNamesOf(n.Child)
	default:
		return nil
	}
}

// executeLocked はロック取得済みのプランノードを実行する。
func (e *Engine) executeLocked(node planner.PlanNode, txnID uint64) (*Result, error) {
	exec, err := e.build(node, txnID)
	if err != nil {
		return nil, err
	}
	defer exec.Close() //nolint:errcheck

	result := &Result{Schema: exec.Schema()}
	for {
		row, err := exec.Next()
		if err != nil {
			return nil, err
		}
		if row == nil {
			break
		}
		result.Rows = append(result.Rows, *row)
	}
	return result, nil
}

// build はプランノードをExecutorの木に変換する。txnIDはInsert/Update/Deleteの各Executorに渡す。
func (e *Engine) build(node planner.PlanNode, txnID uint64) (Executor, error) {
	switch n := node.(type) {
	case *planner.SequentialScanNode:
		return &SeqScanExecutor{repo: e.repo, table: n.Table, schema: n.Schema}, nil
	case *planner.IndexScanNode:
		return &IndexScanExecutor{repo: e.repo, table: n.Table, schema: n.Schema, pkExpr: n.PkExpr}, nil
	case *planner.FilterNode:
		child, err := e.build(n.Child, txnID)
		if err != nil {
			return nil, err
		}
		return &FilterExecutor{child: child, cond: n.Condition}, nil
	case *planner.ProjectionNode:
		child, err := e.build(n.Child, txnID)
		if err != nil {
			return nil, err
		}
		return &ProjectionExecutor{child: child, cols: n.Columns, schema: child.Schema()}, nil
	case *planner.NestedLoopJoinNode:
		left, err := e.build(n.Left, txnID)
		if err != nil {
			return nil, err
		}
		right, err := e.build(n.Right, txnID)
		if err != nil {
			return nil, err
		}
		return &NestedLoopJoinExecutor{repo: e.repo, left: left, right: right, cond: n.Condition}, nil
	case *planner.SortNode:
		child, err := e.build(n.Child, txnID)
		if err != nil {
			return nil, err
		}
		return &SortExecutor{child: child, column: n.Column, desc: n.Desc}, nil
	case *planner.LimitNode:
		child, err := e.build(n.Child, txnID)
		if err != nil {
			return nil, err
		}
		return &LimitExecutor{child: child, count: n.Count}, nil
	case *planner.InsertNode:
		return &InsertExecutor{repo: e.repo, node: n, txnID: txnID}, nil
	case *planner.UpdateNode:
		return &UpdateExecutor{repo: e.repo, node: n, txnID: txnID}, nil
	case *planner.DeleteNode:
		return &DeleteExecutor{repo: e.repo, node: n, txnID: txnID}, nil
	}
	return nil, fmt.Errorf("executor: unsupported plan node %T", node)
}

// ---- SeqScanExecutor ----

type SeqScanExecutor struct {
	repo   TableRepository
	table  string
	schema *types.Schema
	rows   []types.Row
	pos    int
	loaded bool
}

// Next はテーブルの全レコードを1件ずつ返す。最初の呼び出しでストレージから全件読み込む。
func (e *SeqScanExecutor) Next() (*types.Row, error) {
	if !e.loaded {
		rows, err := e.repo.Scan(e.table)
		if err != nil {
			return nil, err
		}
		e.rows = rows
		e.loaded = true
	}
	if e.pos >= len(e.rows) {
		return nil, nil
	}
	row := e.rows[e.pos]
	e.pos++
	return &row, nil
}

func (e *SeqScanExecutor) Schema() *types.Schema { return e.schema }
func (e *SeqScanExecutor) Close() error          { return nil }

// ---- IndexScanExecutor ----

type IndexScanExecutor struct {
	repo   TableRepository
	table  string
	schema *types.Schema
	pkExpr ast.Expression
	done   bool
}

// Next はPKの値でB+Treeを検索して1件だけ返す。2回目以降はnilを返す。
func (e *IndexScanExecutor) Next() (*types.Row, error) {
	if e.done {
		return nil, nil
	}
	e.done = true
	bin, ok := e.pkExpr.(*ast.BinaryExpr)
	if !ok {
		return nil, fmt.Errorf("IndexScan: invalid pk expression")
	}
	pkVal, err := evalLiteral(bin.Right)
	if err != nil {
		return nil, err
	}
	return e.repo.FindByPK(e.table, pkVal)
}

func (e *IndexScanExecutor) Schema() *types.Schema { return e.schema }
func (e *IndexScanExecutor) Close() error          { return nil }

// ---- FilterExecutor ----

type FilterExecutor struct {
	child Executor
	cond  ast.Expression
}

// Next はchildからレコードを1件ずつ受け取り、WHERE条件を満たすレコードだけを返す。
func (e *FilterExecutor) Next() (*types.Row, error) {
	for {
		row, err := e.child.Next()
		if err != nil || row == nil {
			return nil, err
		}
		ok, err := evalCond(e.cond, row, e.child.Schema())
		if err != nil {
			return nil, err
		}
		if ok {
			return row, nil
		}
	}
}

func (e *FilterExecutor) Schema() *types.Schema { return e.child.Schema() }
func (e *FilterExecutor) Close() error          { return e.child.Close() }

// ---- ProjectionExecutor ----

type ProjectionExecutor struct {
	child  Executor
	cols   []ast.Expression
	schema *types.Schema
}

// Next はchildからレコードを1件受け取り、SELECT句で指定されたカラムだけを残して返す。
func (e *ProjectionExecutor) Next() (*types.Row, error) {
	row, err := e.child.Next()
	if err != nil || row == nil {
		return nil, err
	}
	if _, ok := e.cols[0].(*ast.Wildcard); ok {
		return row, nil
	}
	vals := make([]types.Value, len(e.cols))
	for i, col := range e.cols {
		ident, ok := col.(*ast.Identifier)
		if !ok {
			return nil, fmt.Errorf("projection: unsupported expression")
		}
		idx := colIndex(e.child.Schema(), ident.Column)
		if idx < 0 {
			return nil, fmt.Errorf("column %q not found", ident.Column)
		}
		vals[i] = row.Values[idx]
	}
	return &types.Row{Values: vals}, nil
}

func (e *ProjectionExecutor) Schema() *types.Schema { return e.schema }
func (e *ProjectionExecutor) Close() error          { return e.child.Close() }

// ---- NestedLoopJoinExecutor ----

type NestedLoopJoinExecutor struct {
	repo      TableRepository
	left      Executor
	right     Executor
	cond      ast.Expression
	leftRow   *types.Row
	rightRows []types.Row
	rightPos  int
	loaded    bool
}

// Next は左テーブルの各レコードに対して右テーブルの全レコードを突き合わせ、ON条件を満たすレコードを返す。
func (e *NestedLoopJoinExecutor) Next() (*types.Row, error) {
	for {
		if e.leftRow == nil {
			row, err := e.left.Next()
			if err != nil || row == nil {
				return nil, err
			}
			e.leftRow = row
			e.rightRows = nil
			e.rightPos = 0
			e.loaded = false
		}
		// 右のテーブルのすべてのレコードをe.rightRowsに格納
		if !e.loaded {
			for {
				r, err := e.right.Next()
				if err != nil {
					return nil, err
				}
				if r == nil {
					break
				}
				e.rightRows = append(e.rightRows, *r)
			}
			e.loaded = true
		}
		for e.rightPos < len(e.rightRows) {
			right := e.rightRows[e.rightPos]
			e.rightPos++
			joined := joinRows(e.leftRow, &right)
			joinedSchema := joinSchemas(e.left.Schema(), e.right.Schema())
			ok, err := evalCond(e.cond, joined, joinedSchema)
			if err != nil {
				return nil, err
			}
			if ok {
				return joined, nil
			}
		}
		e.leftRow = nil
	}
}

func (e *NestedLoopJoinExecutor) Schema() *types.Schema {
	return joinSchemas(e.left.Schema(), e.right.Schema())
}
func (e *NestedLoopJoinExecutor) Close() error {
	e.left.Close()  //nolint:errcheck
	e.right.Close() //nolint:errcheck
	return nil
}

// ---- SortExecutor ----

type SortExecutor struct {
	child  Executor
	column string
	desc   bool
	rows   []types.Row
	pos    int
	loaded bool
}

// Next は最初の呼び出しでchildから全レコードを引き上げてソートし、1件ずつ返す。
func (e *SortExecutor) Next() (*types.Row, error) {
	if !e.loaded {
		for {
			row, err := e.child.Next()
			if err != nil {
				return nil, err
			}
			if row == nil {
				break
			}
			e.rows = append(e.rows, *row)
		}
		idx := colIndex(e.child.Schema(), e.column)
		sort.SliceStable(e.rows, func(i, j int) bool {
			a := e.rows[i].Values[idx]
			b := e.rows[j].Values[idx]
			cmp := compareValues(a, b)
			if e.desc {
				return cmp > 0
			}
			return cmp < 0
		})
		e.loaded = true
	}
	if e.pos >= len(e.rows) {
		return nil, nil
	}
	row := e.rows[e.pos]
	e.pos++
	return &row, nil
}

func (e *SortExecutor) Schema() *types.Schema { return e.child.Schema() }
func (e *SortExecutor) Close() error          { return e.child.Close() }

// ---- LimitExecutor ----

type LimitExecutor struct {
	child Executor
	count int
	n     int
}

// Next は指定件数に達したらnilを返して終了する。
func (e *LimitExecutor) Next() (*types.Row, error) {
	if e.n >= e.count {
		return nil, nil
	}
	row, err := e.child.Next()
	if err != nil || row == nil {
		return nil, err
	}
	e.n++
	return row, nil
}

func (e *LimitExecutor) Schema() *types.Schema { return e.child.Schema() }
func (e *LimitExecutor) Close() error          { return e.child.Close() }

// ---- InsertExecutor ----

type InsertExecutor struct {
	repo TableRepository
	node  *planner.InsertNode
	txnID uint64
	done  bool
}

// Next は1回だけレコードを挿入してnilを返す。
func (e *InsertExecutor) Next() (*types.Row, error) {
	if e.done {
		return nil, nil
	}
	e.done = true
	vals := make([]types.Value, len(e.node.Values))
	for i, expr := range e.node.Values {
		v, err := evalLiteral(expr)
		if err != nil {
			return nil, err
		}
		vals[i] = v
	}
	row := types.Row{Values: vals}
	return nil, e.repo.Insert(e.node.Table, row, e.txnID)
}

func (e *InsertExecutor) Schema() *types.Schema { return e.node.Schema }
func (e *InsertExecutor) Close() error          { return nil }

// ---- UpdateExecutor ----

type UpdateExecutor struct {
	repo  TableRepository
	node  *planner.UpdateNode
	txnID uint64
	done  bool
}

// Next は1回だけ全件スキャンしてWHERE条件に一致するレコードを更新してnilを返す。
func (e *UpdateExecutor) Next() (*types.Row, error) {
	if e.done {
		return nil, nil
	}
	e.done = true
	rows, err := e.repo.Scan(e.node.Table)
	if err != nil {
		return nil, err
	}
	pkIdx := e.node.Schema.PrimaryKeyIndex()
	for _, row := range rows {
		if e.node.Where != nil {
			ok, err := evalCond(e.node.Where, &row, e.node.Schema)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
		}
		updated := make([]types.Value, len(row.Values))
		copy(updated, row.Values)
		for _, assign := range e.node.Assignments {
			idx := colIndex(e.node.Schema, assign.Column)
			if idx < 0 {
				return nil, fmt.Errorf("column %q not found", assign.Column)
			}
			v, err := evalLiteral(assign.Value)
			if err != nil {
				return nil, err
			}
			updated[idx] = v
		}
		pk := row.Values[pkIdx]
		if err := e.repo.Update(e.node.Table, pk, types.Row{Values: updated}, e.txnID); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (e *UpdateExecutor) Schema() *types.Schema { return e.node.Schema }
func (e *UpdateExecutor) Close() error          { return nil }

// ---- DeleteExecutor ----

type DeleteExecutor struct {
	repo  TableRepository
	node  *planner.DeleteNode
	txnID uint64
	done  bool
}

// Next は1回だけ全件スキャンしてWHERE条件に一致するレコードを削除してnilを返す。
func (e *DeleteExecutor) Next() (*types.Row, error) {
	if e.done {
		return nil, nil
	}
	e.done = true
	rows, err := e.repo.Scan(e.node.Table)
	if err != nil {
		return nil, err
	}
	pkIdx := e.node.Schema.PrimaryKeyIndex()
	for _, row := range rows {
		if e.node.Where != nil {
			ok, err := evalCond(e.node.Where, &row, e.node.Schema)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
		}
		pk := row.Values[pkIdx]
		if err := e.repo.Delete(e.node.Table, pk, e.txnID); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (e *DeleteExecutor) Schema() *types.Schema { return e.node.Schema }
func (e *DeleteExecutor) Close() error          { return nil }

// ---- ユーティリティ ----

// evalLiteral はリテラル式をtypes.Valueに変換する。
func evalLiteral(expr ast.Expression) (types.Value, error) {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return types.IntValue{V: e.Value}, nil
	case *ast.StringLiteral:
		return types.StringValue{V: e.Value}, nil
	case *ast.BoolLiteral:
		return types.BoolValue{V: e.Value}, nil
	case *ast.NullLiteral:
		return types.NullValue{}, nil
	}
	return nil, fmt.Errorf("evalLiteral: unsupported expression %T", expr)
}

// evalCond はWHERE条件の式を評価して、レコードが条件を満たすかどうかを返す。
func evalCond(expr ast.Expression, row *types.Row, schema *types.Schema) (bool, error) {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		return evalBinary(e, row, schema)
	case *ast.UnaryExpr:
		if e.Operator == "NOT" {
			ok, err := evalCond(e.Operand, row, schema)
			return !ok, err
		}
	case *ast.IsNullExpr:
		v, err := evalValue(e.Operand, row, schema)
		if err != nil {
			return false, err
		}
		_, isNull := v.(types.NullValue)
		return isNull, nil
	case *ast.IsNotNullExpr:
		v, err := evalValue(e.Operand, row, schema)
		if err != nil {
			return false, err
		}
		_, isNull := v.(types.NullValue)
		return !isNull, nil
	}
	return false, fmt.Errorf("evalCond: unsupported %T", expr)
}

// evalBinary は二項演算式を評価する。AND/ORは短絡評価、比較演算子はcompareValuesの結果を使う。
func evalBinary(e *ast.BinaryExpr, row *types.Row, schema *types.Schema) (bool, error) {
	if e.Operator == ast.OpAND {
		l, err := evalCond(e.Left, row, schema)
		if err != nil || !l {
			return false, err
		}
		return evalCond(e.Right, row, schema)
	}
	if e.Operator == ast.OpOR {
		l, err := evalCond(e.Left, row, schema)
		if err != nil || l {
			return l, err
		}
		return evalCond(e.Right, row, schema)
	}
	lv, err := evalValue(e.Left, row, schema)
	if err != nil {
		return false, err
	}
	rv, err := evalValue(e.Right, row, schema)
	if err != nil {
		return false, err
	}
	cmp := compareValues(lv, rv)
	switch e.Operator {
	case ast.OpEQ:
		return cmp == 0, nil
	case ast.OpNEQ:
		return cmp != 0, nil
	case ast.OpLT:
		return cmp < 0, nil
	case ast.OpGT:
		return cmp > 0, nil
	case ast.OpLTE:
		return cmp <= 0, nil
	case ast.OpGTE:
		return cmp >= 0, nil
	}
	return false, fmt.Errorf("evalBinary: unsupported operator %v", e.Operator)
}

// evalValue は式をレコードのコンテキストで評価してtypes.Valueを返す。Identifierの場合はレコードから値を取り出す。
func evalValue(expr ast.Expression, row *types.Row, schema *types.Schema) (types.Value, error) {
	switch e := expr.(type) {
	case *ast.Identifier:
		idx := colIndex(schema, e.Column)
		if idx < 0 {
			return nil, fmt.Errorf("column %q not found", e.Column)
		}
		return row.Values[idx], nil
	case *ast.IntLiteral:
		return types.IntValue{V: e.Value}, nil
	case *ast.StringLiteral:
		return types.StringValue{V: e.Value}, nil
	case *ast.BoolLiteral:
		return types.BoolValue{V: e.Value}, nil
	case *ast.NullLiteral:
		return types.NullValue{}, nil
	}
	return nil, fmt.Errorf("evalValue: unsupported %T", expr)
}

// compareValues は2つの値を比較して-1/0/1を返す。型不一致やNULLのケースは未対応。
func compareValues(a, b types.Value) int {
	switch av := a.(type) {
	case types.IntValue:
		bv := b.(types.IntValue)
		if av.V < bv.V {
			return -1
		} else if av.V > bv.V {
			return 1
		}
		return 0
	case types.StringValue:
		bv := b.(types.StringValue)
		if av.V < bv.V {
			return -1
		} else if av.V > bv.V {
			return 1
		}
		return 0
	}
	return 0
}

// colIndex はスキーマからカラム名の位置を返す。見つからない場合は-1を返す。
func colIndex(schema *types.Schema, name string) int {
	for i, col := range schema.Columns {
		if col.Name == name {
			return i
		}
	}
	return -1
}

// joinRows は2つのレコードのValuesを結合して1つのレコードにする。
func joinRows(a, b *types.Row) *types.Row {
	vals := make([]types.Value, len(a.Values)+len(b.Values))
	copy(vals, a.Values)
	copy(vals[len(a.Values):], b.Values)
	return &types.Row{Values: vals}
}

// joinSchemas は2つのスキーマのカラムを結合して1つのスキーマにする。
func joinSchemas(a, b *types.Schema) *types.Schema {
	cols := make([]types.Column, len(a.Columns)+len(b.Columns))
	copy(cols, a.Columns)
	copy(cols[len(a.Columns):], b.Columns)
	return &types.Schema{TableName: a.TableName + "_" + b.TableName, Columns: cols}
}
