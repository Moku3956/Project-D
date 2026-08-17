package txn

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Moku3956/Project-D/storage/wal"
)

const lockTimeout = 5 * time.Second

type State int

const (
	StateActive    State = iota
	StateCommitted
	StateAborted
)

// Txn はトランザクション1件を表す。
type Txn struct {
	ID    uint64
	State State
	locks []string // ロックを取得済みのテーブル名
}

// Manager はトランザクションのライフサイクルとテーブルロックを管理する。
type Manager struct {
	wm       *wal.WALManager
	nextTxnID atomic.Uint64

	mu         sync.Mutex
	tableLocks map[string]*sync.RWMutex
}

func NewManager(wm *wal.WALManager) *Manager {
	return &Manager{
		wm:         wm,
		tableLocks: make(map[string]*sync.RWMutex),
	}
}

// Begin は新しいトランザクションを開始する。
func (m *Manager) Begin() *Txn {
	id := m.nextTxnID.Add(1)
	return &Txn{ID: id, State: StateActive}
}

// RLock はテーブルの読み取りロックを取得する。
func (m *Manager) RLock(txn *Txn, table string) error {
	mu := m.tableMu(table)
	done := make(chan struct{})
	go func() {
		mu.RLock()
		close(done)
	}()
	select {
	case <-done:
		txn.locks = append(txn.locks, "r:"+table)
		return nil
	case <-time.After(lockTimeout):
		return fmt.Errorf("txn %d: lock timeout on table %q", txn.ID, table)
	}
}

// Lock はテーブルの書き込みロックを取得する。
func (m *Manager) Lock(txn *Txn, table string) error {
	mu := m.tableMu(table)
	done := make(chan struct{})
	go func() {
		mu.Lock()
		close(done)
	}()
	select {
	case <-done:
		txn.locks = append(txn.locks, "w:"+table)
		return nil
	case <-time.After(lockTimeout):
		return fmt.Errorf("txn %d: lock timeout on table %q", txn.ID, table)
	}
}

// Commit はWALにCOMMITレコードを書いてfsyncし、ロックを解放する。
func (m *Manager) Commit(txn *Txn) error {
	if txn.State != StateActive {
		return fmt.Errorf("txn %d: not active", txn.ID)
	}
	_, err := m.wm.Append(&wal.LogRecord{
		TxnID: txn.ID,
		Op:    wal.OpCommit,
	})
	if err != nil {
		return err
	}
	if err := m.wm.Flush(); err != nil {
		return err
	}
	txn.State = StateCommitted
	m.unlock(txn)
	return nil
}

// Rollback はdirtyページを破棄してロックを解放する。
func (m *Manager) Rollback(txn *Txn) error {
	if txn.State != StateActive {
		return fmt.Errorf("txn %d: not active", txn.ID)
	}
	_, err := m.wm.Append(&wal.LogRecord{
		TxnID: txn.ID,
		Op:    wal.OpAbort,
	})
	if err != nil {
		return err
	}
	txn.State = StateAborted
	m.unlock(txn)
	return nil
}

func (m *Manager) unlock(txn *Txn) {
	for _, entry := range txn.locks {
		table := entry[2:]
		mu := m.tableMu(table)
		if entry[:2] == "r:" {
			mu.RUnlock()
		} else {
			mu.Unlock()
		}
	}
	txn.locks = nil
}

func (m *Manager) tableMu(table string) *sync.RWMutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tableLocks[table]; !ok {
		m.tableLocks[table] = &sync.RWMutex{}
	}
	return m.tableLocks[table]
}
