package txn

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Moku3956/Project-D/storage/wal"
)

// ---- ヘルパー ----

func newManager(t *testing.T) *Manager {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wal.log")
	wm, err := wal.NewWALManager(path)
	if err != nil {
		t.Fatalf("NewWALManager error: %v", err)
	}
	return NewManager(wm)
}

// ---- 正常系 ----

func TestBegin(t *testing.T) {
	m := newManager(t)
	txn := m.Begin()

	if txn.State != StateActive {
		t.Errorf("State = %v, want StateActive", txn.State)
	}
	if txn.ID == 0 {
		t.Error("ID が 0 になっている")
	}
}

func TestTxnIDIncrement(t *testing.T) {
	m := newManager(t)
	t1 := m.Begin()
	t2 := m.Begin()

	if t1.ID == t2.ID {
		t.Errorf("TxnID が重複している: %d", t1.ID)
	}
	if t2.ID != t1.ID+1 {
		t.Errorf("TxnID = %d, want %d", t2.ID, t1.ID+1)
	}
}

func TestCommit(t *testing.T) {
	m := newManager(t)
	txn := m.Begin()

	if err := m.Commit(txn); err != nil {
		t.Fatalf("Commit error: %v", err)
	}
	if txn.State != StateCommitted {
		t.Errorf("State = %v, want StateCommitted", txn.State)
	}
	if len(txn.locks) != 0 {
		t.Errorf("Commit後にロックが残っている: %v", txn.locks)
	}
}

func TestRollback(t *testing.T) {
	m := newManager(t)
	txn := m.Begin()

	if err := m.Rollback(txn); err != nil {
		t.Fatalf("Rollback error: %v", err)
	}
	if txn.State != StateAborted {
		t.Errorf("State = %v, want StateAborted", txn.State)
	}
	if len(txn.locks) != 0 {
		t.Errorf("Rollback後にロックが残っている: %v", txn.locks)
	}
}

func TestRLockAndLock(t *testing.T) {
	m := newManager(t)
	txn := m.Begin()

	if err := m.RLock(txn, "users"); err != nil {
		t.Fatalf("RLock error: %v", err)
	}
	if err := m.RLock(txn, "orders"); err != nil {
		t.Fatalf("RLock error: %v", err)
	}
	if err := m.Commit(txn); err != nil {
		t.Fatalf("Commit error: %v", err)
	}

	txn2 := m.Begin()
	if err := m.Lock(txn2, "users"); err != nil {
		t.Fatalf("Lock error: %v", err)
	}
	if err := m.Commit(txn2); err != nil {
		t.Fatalf("Commit error: %v", err)
	}
}

// ---- 異常系 ----

func TestCommitNotActive(t *testing.T) {
	m := newManager(t)
	txn := m.Begin()
	if err := m.Commit(txn); err != nil {
		t.Fatalf("Commit error: %v", err)
	}

	err := m.Commit(txn)
	if err == nil {
		t.Fatal("コミット済みへのCommitでエラーが期待されたがnil")
	}
}

func TestRollbackNotActive(t *testing.T) {
	m := newManager(t)
	txn := m.Begin()
	if err := m.Commit(txn); err != nil {
		t.Fatalf("Commit error: %v", err)
	}

	err := m.Rollback(txn)
	if err == nil {
		t.Fatal("コミット済みへのRollbackでエラーが期待されたがnil")
	}
}

func TestLockTimeout(t *testing.T) {
	m := newManager(t)

	// txn1が書き込みロックを取得したまま保持する
	txn1 := m.Begin()
	if err := m.Lock(txn1, "users"); err != nil {
		t.Fatalf("Lock error: %v", err)
	}

	// txn2が同じテーブルの書き込みロックを取得しようとしてタイムアウトする
	txn2 := m.Begin()

	// lockTimeoutを短くするため、内部定数の代わりに直接goroutineで試みる
	done := make(chan error, 1)
	go func() {
		done <- m.Lock(txn2, "users")
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("タイムアウトエラーが期待されたがnil")
		}
	case <-time.After(lockTimeout + time.Second):
		t.Fatal("タイムアウトより先にテストがタイムアウトした")
	}

	if err := m.Commit(txn1); err != nil {
		t.Fatalf("Commit error: %v", err)
	}
}

func TestRLockConcurrent(t *testing.T) {
	m := newManager(t)

	// 複数のトランザクションが同時に読み取りロックを取得できる
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			txn := m.Begin()
			if err := m.RLock(txn, "users"); err != nil {
				t.Errorf("RLock error: %v", err)
				return
			}
			if err := m.Commit(txn); err != nil {
				t.Errorf("Commit error: %v", err)
			}
		}()
	}
	wg.Wait()
}
