package infrastructure

import (
	"fmt"
	"sync"

	"github.com/Moku3956/Project-D/storage/btree"
	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/types"
)

// BTreeRepository は TableRepository を単一の B+Tree で実装する。
// 全テーブルのデータが1ファイルに格納され、tableID プレフィックスで識別される。
type BTreeRepository struct {
	mu      sync.RWMutex
	bt      *btree.BTree
	schemas map[string]*types.Schema
}

func NewBTreeRepository(disk *page.DiskManager) (*BTreeRepository, error) {
	bt, err := btree.NewBTree(disk)
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

func (r *BTreeRepository) FindByPK(table string, pk types.Value) (*types.Row, error) {
	schema, err := r.schema(table)
	if err != nil {
		return nil, err
	}
	return r.bt.Search(schema.TableID, pk, schema)
}

func (r *BTreeRepository) Scan(table string) ([]types.Row, error) {
	schema, err := r.schema(table)
	if err != nil {
		return nil, err
	}
	return r.bt.Scan(schema.TableID, schema)
}

func (r *BTreeRepository) Insert(table string, row types.Row) error {
	schema, err := r.schema(table)
	if err != nil {
		return err
	}
	pk := row.Values[schema.PrimaryKeyIndex()]
	return r.bt.Insert(schema.TableID, pk, row, schema)
}

func (r *BTreeRepository) Update(table string, pk types.Value, row types.Row) error {
	schema, err := r.schema(table)
	if err != nil {
		return err
	}
	if err := r.bt.Delete(schema.TableID, pk); err != nil {
		return err
	}
	return r.bt.Insert(schema.TableID, pk, row, schema)
}

func (r *BTreeRepository) Delete(table string, pk types.Value) error {
	schema, err := r.schema(table)
	if err != nil {
		return err
	}
	return r.bt.Delete(schema.TableID, pk)
}

func (r *BTreeRepository) schema(table string) (*types.Schema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.schemas[table]
	if !ok {
		return nil, fmt.Errorf("table %q is not open", table)
	}
	return s, nil
}
