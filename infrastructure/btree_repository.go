package infrastructure

import (
	"fmt"
	"sync"

	"github.com/Moku3956/Project-D/storage/btree"
	"github.com/Moku3956/Project-D/storage/buffer"
	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
	"github.com/Moku3956/Project-D/types"
)

// BTreeRepository は TableRepository を単一の B+Tree で実装する。
// 全テーブルのデータが1ファイルに格納され、tableID プレフィックスで識別される。
type BTreeRepository struct {
	mu      sync.RWMutex
	bt      *btree.BTree
	schemas map[string]*types.Schema
}

// NewBTreeRepository はDiskManagerを受け取りB+Treeを初期化してBTreeRepositoryを返す。
func NewBTreeRepository(disk *page.DiskManager, bp *buffer.BufferPool, wm *wal.WALManager) (*BTreeRepository, error) {
	bt, err := btree.NewBTree(disk, bp, wm)
	if err != nil {
		return nil, err
	}
	return &BTreeRepository{
		bt:      bt,
		schemas: make(map[string]*types.Schema),
	}, nil
}

// OpenTable はテーブルのスキーマを登録する。起動時・CREATE TABLE後に呼ぶ。
func (r *BTreeRepository) OpenTable(table string, schema *types.Schema) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schemas[table] = schema
	return nil
}

// FindByPK はPKでB+Treeを検索してレコード1件を返す。見つからない場合はnilを返す。
func (r *BTreeRepository) FindByPK(table string, pk types.Value) (*types.Row, error) {
	schema, err := r.schema(table)
	if err != nil {
		return nil, err
	}
	return r.bt.Search(schema.TableID, pk, schema)
}

// Scan はテーブルの全レコードを返す。
func (r *BTreeRepository) Scan(table string) ([]types.Row, error) {
	schema, err := r.schema(table)
	if err != nil {
		return nil, err
	}
	return r.bt.Scan(schema.TableID, schema)
}

// Insert はレコードのPKを取り出してB+Treeに挿入する。
func (r *BTreeRepository) Insert(table string, row types.Row, txnID uint64) error {
	schema, err := r.schema(table)
	if err != nil {
		return err
	}
	pk := row.Values[schema.PrimaryKeyIndex()]
	return r.bt.Insert(schema.TableID, pk, row, schema, txnID)
}

// Update は既存レコードをDeleteしてから新しいレコードをInsertする。B+TreeにUpdateがないため。
func (r *BTreeRepository) Update(table string, pk types.Value, row types.Row, txnID uint64) error {
	schema, err := r.schema(table)
	if err != nil {
		return err
	}
	if err := r.bt.Delete(schema.TableID, pk, txnID); err != nil {
		return err
	}
	return r.bt.Insert(schema.TableID, pk, row, schema, txnID)
}

// Delete はPKでB+Treeからレコードを削除する。
func (r *BTreeRepository) Delete(table string, pk types.Value, txnID uint64) error {
	schema, err := r.schema(table)
	if err != nil {
		return err
	}
	return r.bt.Delete(schema.TableID, pk, txnID)
}

// schema はテーブル名からキャッシュ済みのスキーマを返す。未登録の場合はエラー。
func (r *BTreeRepository) schema(table string) (*types.Schema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.schemas[table]
	if !ok {
		return nil, fmt.Errorf("table %q is not open", table)
	}
	return s, nil
}

// Schema はテーブル名からキャッシュ済みのスキーマを返す。db-internal-appが
// DumpTree呼び出し時にTableIDを取得するために使う公開版(schemaのエクスポート)。
func (r *BTreeRepository) Schema(table string) (*types.Schema, error) {
	return r.schema(table)
}

// BTree は内部で保持しているB+Treeを返す。db-internal-appのようにストレージ
// 内部を読み取り専用で可視化する用途のために公開する(通常の呼び出し元は
// FindByPK/Scan/Insert/Update/Deleteを使うべきで、これらを経由すべきではない)。
func (r *BTreeRepository) BTree() *btree.BTree {
	return r.bt
}
