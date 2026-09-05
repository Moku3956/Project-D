// Package dbsession はdb-internal-appの1セッション分のProject-Dインスタンスを
// 管理する。
//
// sql-monsterと違い、db-internal-appはSQLの内部処理を可視化することが目的の
// ため、公開用の`client`パッケージ(最終的なResultしか返さない)は使わず、
// catalog/sql/planner/executor/storage/btreeを直接importして組み立てる
// (db-internal-app/docs/spec.md「アーキテクチャ」参照)。ここでの配線は
// client.Open/client.DB.Execとほぼ同じだが、DumpTree呼び出しのために
// infrastructure.BTreeRepositoryへの型付き参照を保持し続ける点が異なる
// (client.DBはexecutor.TableRepositoryインターフェース越しにしか repo を
// 持たないため、DumpTreeを呼び出せない)。
package dbsession

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Moku3956/Project-D/catalog"
	"github.com/Moku3956/Project-D/executor"
	"github.com/Moku3956/Project-D/infrastructure"
	"github.com/Moku3956/Project-D/sql/ast"
	"github.com/Moku3956/Project-D/sql/parser"
	"github.com/Moku3956/Project-D/sql/planner"
	"github.com/Moku3956/Project-D/storage/btree"
	"github.com/Moku3956/Project-D/storage/buffer"
	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
	"github.com/Moku3956/Project-D/txn"
	"github.com/Moku3956/Project-D/types"
)

// defaultBufferPoolSize はclient.Openと同じデフォルト値。
const defaultBufferPoolSize = 1000

// Session は1セッション分のDBインスタンス。ディレクトリごとに独立した
// DiskManager/WAL/Catalogを持つ(db-internal-app/docs/spec.md「セッションの
// 永続性」参照)。
type Session struct {
	disk *page.DiskManager
	wm   *wal.WALManager
	cat  *catalog.Catalog
	repo *infrastructure.BTreeRepository
	pl   *planner.Planner
	eng  *executor.Engine
	txnM *txn.Manager
}

// Open はdir配下にDBファイル一式を作成(または開く)する。起動時にクラッシュ
// 復旧(WAL Redo)を行う。
func Open(dir string) (*Session, error) {
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

	return &Session{
		disk: dm,
		wm:   wm,
		cat:  cat,
		repo: repo,
		pl:   planner.NewPlanner(cat),
		eng:  executor.NewEngine(repo, cat, txnMgr),
		txnM: txnMgr,
	}, nil
}

// Close はディスク・WALのファイルハンドルを閉じる。
func (s *Session) Close() error {
	if err := s.wm.Close(); err != nil {
		return err
	}
	return s.disk.Close()
}

// Exec はSQLを1文=1トランザクションの自動コミットで実行する。
func (s *Session) Exec(sql string) (*executor.Result, error) {
	stmt, err := parser.Parse(sql)
	if err != nil {
		return nil, err
	}
	// ユーザーが書くSQLは常に短い素のPK値でよいようにする(パディングは
	// db-internal-app側の便宜機能。padding.go参照)。対象テーブルが存在しない
	// 場合は素通しし、後続のPlanで通常のエラーとして扱わせる。
	if table := targetTableName(stmt); table != "" {
		if schema, err := s.cat.GetSchema(table); err == nil {
			padPKLiterals(stmt, schema)
		}
	}
	node, err := s.pl.Plan(stmt)
	if err != nil {
		return nil, err
	}
	return s.eng.Execute(node)
}

// targetTableName はSELECT/INSERT/UPDATE/DELETEの対象テーブル名を返す
// (それ以外の文は空文字)。
func targetTableName(stmt ast.Statement) string {
	switch s := stmt.(type) {
	case *ast.SelectStatement:
		return s.Table
	case *ast.InsertStatement:
		return s.Table
	case *ast.UpdateStatement:
		return s.Table
	case *ast.DeleteStatement:
		return s.Table
	default:
		return ""
	}
}

// TableInfo は1テーブルのメタ情報(テーブル切り替えUI用)。
type TableInfo struct {
	Name    string
	Columns []string
}

// Tables はセッション内に存在する全テーブルの名前・カラム一覧を、名前の
// 辞書順で返す(catalog.TableNamesはmapのイテレーション順で不定なため)。
func (s *Session) Tables() ([]TableInfo, error) {
	names := s.cat.TableNames()
	sort.Strings(names)
	infos := make([]TableInfo, len(names))
	for i, name := range names {
		schema, err := s.cat.GetSchema(name)
		if err != nil {
			return nil, fmt.Errorf("tables: %w", err)
		}
		cols := make([]string, len(schema.Columns))
		for j, c := range schema.Columns {
			cols[j] = c.Name
		}
		infos[i] = TableInfo{Name: name, Columns: cols}
	}
	return infos, nil
}

// DumpTree はB+Tree全体を現在の状態でスナップショットする(storage/btree.BTree.
// DumpTree参照)。tableは「その名前のテーブルが存在するか」の検証にのみ使う。
// 1つの物理木を全テーブルで共有しているため、返る内容はtableによらず常に同じ
// (全テーブルの行を含み、各行がどのテーブルのものかはPageSnapshot.RowTablesで
// 分かる)。tableがOpenTableされていない場合はエラーを返す。
func (s *Session) DumpTree(table string) (*btree.TreeSnapshot, error) {
	if _, err := s.repo.Schema(table); err != nil {
		return nil, fmt.Errorf("dump tree: %w", err)
	}

	names := s.cat.TableNames()
	schemas := make(map[uint32]*types.Schema, len(names))
	for _, name := range names {
		schema, err := s.cat.GetSchema(name)
		if err != nil {
			return nil, fmt.Errorf("dump tree: %w", err)
		}
		schemas[schema.TableID] = schema
	}
	return s.repo.BTree().DumpTree(schemas)
}
