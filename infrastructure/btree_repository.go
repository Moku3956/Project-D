package infrastructure

import (
	"fmt"
	"sync"

	"github.com/Moku3956/Project-D/storage/btree"
	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/types"
)

// BTreeRepository は TableRepository を B+Tree で実装する。
type BTreeRepository struct {
	mu     sync.RWMutex
	disk   *page.DiskManager
	trees  map[string]*btree.BTree
}

func NewBTreeRepository(disk *page.DiskManager) *BTreeRepository {
	return &BTreeRepository{
		disk:  disk,
		trees: make(map[string]*btree.BTree),
	}
}

// OpenTable はテーブル用のB+Treeを初期化する。起動時に呼ぶ。
func (r *BTreeRepository) OpenTable(table string, schema *types.Schema) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	bt, err := btree.NewBTree(r.disk, schema)
	if err != nil {
		return err
	}
	r.trees[table] = bt
	return nil
}

func (r *BTreeRepository) FindByPK(table string, pk types.Value) (*types.Row, error) {
	bt, err := r.tree(table)
	if err != nil {
		return nil, err
	}
	return bt.Search(pk)
}

func (r *BTreeRepository) Scan(table string) ([]types.Row, error) {
	bt, err := r.tree(table)
	if err != nil {
		return nil, err
	}
	return bt.Scan()
}

func (r *BTreeRepository) Insert(table string, row types.Row) error {
	bt, err := r.tree(table)
	if err != nil {
		return err
	}
	schema := bt.Schema()
	pk := row.Values[schema.PrimaryKeyIndex()]
	return bt.Insert(pk, row)
}

func (r *BTreeRepository) Update(table string, pk types.Value, row types.Row) error {
	bt, err := r.tree(table)
	if err != nil {
		return err
	}
	if err := bt.Delete(pk); err != nil {
		return err
	}
	return bt.Insert(pk, row)
}

func (r *BTreeRepository) Delete(table string, pk types.Value) error {
	bt, err := r.tree(table)
	if err != nil {
		return err
	}
	return bt.Delete(pk)
}

func (r *BTreeRepository) tree(table string) (*btree.BTree, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bt, ok := r.trees[table]
	if !ok {
		return nil, fmt.Errorf("table %q is not open", table)
	}
	return bt, nil
}
