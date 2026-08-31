package txn

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Moku3956/Project-D/storage/buffer"
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
	wm        *wal.WALManager
	bp        *buffer.BufferPool
	nextTxnID atomic.Uint64

	mu         sync.Mutex // muはテーブルマップ自体をロック
	tableLocks map[string]*sync.RWMutex // tableLocksはテーブルマップにアクセスをして、ロック
}

func NewManager(wm *wal.WALManager, bp *buffer.BufferPool) *Manager {
	return &Manager{
		wm:         wm,
		bp:         bp,
		tableLocks: make(map[string]*sync.RWMutex),
	}
}

// Begin は新しいトランザクションを開始する。
func (m *Manager) Begin() *Txn {
	// 他のゴルーチンがnextTxtIDを同時にインクリメントしないように
	id := m.nextTxnID.Add(1)
	return &Txn{ID: id, State: StateActive}
}

// RLock はテーブルの読み取りロックを取得する。txnが既にこのテーブルの読み取りまたは
// 書き込みロックを保持していれば、再取得せずそのまま成功する(再入可能)。
func (m *Manager) RLock(txn *Txn, table string) error {
	if txn.holds("r:"+table) || txn.holds("w:"+table) {
		return nil
	}
	mu := m.tableMu(table)
	done := make(chan struct{})
	// ロック処理をメインと分けて処理する。タイムアウト計測を行うため。
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

// Lock はテーブルの書き込みロックを取得する。txnが既にこのテーブルの書き込みロックを
// 保持していれば、再取得せずそのまま成功する(再入可能)。読み取りロックからの
// アップグレードは未対応(自己デッドロックするため、あればエラーを返す)。
func (m *Manager) Lock(txn *Txn, table string) error {
	if txn.holds("w:" + table) {
		return nil
	}
	if txn.holds("r:" + table) {
		return fmt.Errorf("txn %d: cannot upgrade read lock to write lock on table %q", txn.ID, table)
	}
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

// holds はtxnが指定のロックエントリ("r:table"や"w:table")を既に保持しているかを返す。
func (txn *Txn) holds(entry string) bool {
	for _, l := range txn.locks {
		if l == entry {
			return true
		}
	}
	return false
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
	// このトランザクションが汚したページだけをディスクへ反映する(No-Force。他txnの未コミットdirtyには触れない)。
	if err := m.bp.FlushAll(map[uint64]bool{txn.ID: true}); err != nil {
		return err
	}
	txn.State = StateCommitted
	m.unlock(txn)
	return nil
}

// Rollback はバッファプール上のdirtyページを破棄してロックを解放する。
// wmのbufにはabortログが積まれる。
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
	if err := m.bp.DiscardTxn(txn.ID); err != nil {
		return err
	}
	txn.State = StateAborted
	m.unlock(txn)
	return nil
}

// 読み取りか書き込みかを判断し、ロックを開放する
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

// tableLocksマップ自体をロックする
func (m *Manager) tableMu(table string) *sync.RWMutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tableLocks[table]; !ok {
		m.tableLocks[table] = &sync.RWMutex{}
	}
	return m.tableLocks[table]
}
