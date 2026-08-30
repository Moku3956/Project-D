// Package client はProject-Dを外部のGoプログラムから使うための公開インターフェース。
// executor/planner/txn などの内部パッケージはここでラップし、利用側には公開しない。
package client

import (
	"os"
	"path/filepath"

	"github.com/Moku3956/Project-D/catalog"
	"github.com/Moku3956/Project-D/executor"
	"github.com/Moku3956/Project-D/infrastructure"
	"github.com/Moku3956/Project-D/sql/parser"
	"github.com/Moku3956/Project-D/sql/planner"
	"github.com/Moku3956/Project-D/storage/btree"
	"github.com/Moku3956/Project-D/storage/buffer"
	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
	"github.com/Moku3956/Project-D/txn"
)

// defaultBufferPoolSize はバッファプールに保持するページ数の上限のデフォルト値。
const defaultBufferPoolSize = 1000

// DB はProject-Dの1インスタンスへのハンドル。
type DB struct {
	disk *page.DiskManager
	wm   *wal.WALManager
	cat  *catalog.Catalog
	pl   *planner.Planner
	eng  *executor.Engine
	txnM *txn.Manager
}

// Open はdirにある(なければ新規作成する)DiskManager/WAL/Catalogを使ってDBを開く。
// 起動時にクラッシュ復旧(WAL Redo)を行う。
func Open(dir string) (*DB, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	dm, err := page.NewDiskManager(filepath.Join(dir, "mydb.db"))
	if err != nil {
		return nil, err
	}
	wm, err := wal.NewWALManager(filepath.Join(dir, "mydb.wal"))
	if err != nil {
		return nil, err
	}
	if err := btree.Recover(dm, wm); err != nil {
		return nil, err
	}

	cat, err := catalog.NewCatalog(filepath.Join(dir, "catalog.json"))
	if err != nil {
		return nil, err
	}

	bp := buffer.NewBufferPool(dm, wm, defaultBufferPoolSize)

	repo, err := infrastructure.NewBTreeRepository(dm, bp, wm)
	if err != nil {
		return nil, err
	}

	// カタログ上の全テーブルをrepositoryに登録する
	for _, name := range cat.TableNames() {
		schema, err := cat.GetSchema(name)
		if err != nil {
			return nil, err
		}
		if err := repo.OpenTable(name, schema); err != nil {
			return nil, err
		}
	}

	txnMgr := txn.NewManager(wm, bp)

	return &DB{
		disk: dm,
		wm:   wm,
		cat:  cat,
		pl:   planner.NewPlanner(cat),
		eng:  executor.NewEngine(repo, cat, txnMgr),
		txnM: txnMgr,
	}, nil
}

// Close はディスク・WALのファイルハンドルを閉じる。
func (db *DB) Close() error {
	if err := db.wm.Close(); err != nil {
		return err
	}
	return db.disk.Close()
}

// Exec はSQLを1文=1トランザクションの自動コミットで実行する。
func (db *DB) Exec(sql string) (*Result, error) {
	node, err := db.plan(sql)
	if err != nil {
		return nil, err
	}
	res, err := db.eng.Execute(node)
	if err != nil {
		return nil, err
	}
	return &Result{Result: res, Scan: scanKindOf(node)}, nil
}

// Begin は明示的なトランザクションを開始する。
func (db *DB) Begin() *Tx {
	return &Tx{db: db, txn: db.txnM.Begin()}
}

func (db *DB) plan(sql string) (planner.PlanNode, error) {
	stmt, err := parser.Parse(sql)
	if err != nil {
		return nil, err
	}
	return db.pl.Plan(stmt)
}

// Tx はBeginで開始した、呼び出し元が寿命を管理するトランザクション。
type Tx struct {
	db  *DB
	txn *txn.Txn
}

// Exec はこのトランザクションの中でSQLを1文実行する。コミット/ロールバックはしない。
func (tx *Tx) Exec(sql string) (*Result, error) {
	node, err := tx.db.plan(sql)
	if err != nil {
		return nil, err
	}
	res, err := tx.db.eng.ExecuteInTxn(tx.txn, node)
	if err != nil {
		return nil, err
	}
	return &Result{Result: res, Scan: scanKindOf(node)}, nil
}

// Commit はこのトランザクションをコミットする。
func (tx *Tx) Commit() error {
	return tx.db.txnM.Commit(tx.txn)
}

// Rollback はこのトランザクションをロールバックする。
func (tx *Tx) Rollback() error {
	return tx.db.txnM.Rollback(tx.txn)
}

// Result はSQL実行の結果。executor.Resultに、実行計画がどのスキャン戦略を
// 選んだかの情報を added する。
type Result struct {
	*executor.Result
	Scan ScanKind
}

// ScanKind はプランが選んだスキャン戦略。
type ScanKind int

const (
	// ScanNone はスキャンを伴わないノード(INSERT/DDL/BEGIN等)を表す。
	ScanNone ScanKind = iota
	// ScanIndex はプラン中の全スキャンがIndexScan(PKの等値検索)だったことを表す。
	ScanIndex
	// ScanSequential はプラン中に1つでもSequentialScan(全件走査)が含まれることを表す。
	ScanSequential
)

// scanKindOf はプランノードの木を辿り、使われたスキャン戦略を判定する。
func scanKindOf(node planner.PlanNode) ScanKind {
	hasIndex, hasSeq := false, false
	var walk func(planner.PlanNode)
	walk = func(n planner.PlanNode) {
		switch v := n.(type) {
		case *planner.IndexScanNode:
			hasIndex = true
		case *planner.SequentialScanNode:
			hasSeq = true
		case *planner.FilterNode:
			walk(v.Child)
		case *planner.ProjectionNode:
			walk(v.Child)
		case *planner.SortNode:
			walk(v.Child)
		case *planner.LimitNode:
			walk(v.Child)
		case *planner.NestedLoopJoinNode:
			walk(v.Left)
			walk(v.Right)
		}
	}
	walk(node)
	switch {
	case hasSeq:
		return ScanSequential
	case hasIndex:
		return ScanIndex
	default:
		return ScanNone
	}
}
