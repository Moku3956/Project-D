package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestSessionStore(t *testing.T) *sessionStore {
	t.Helper()
	dir := t.TempDir()
	st := newSessionStore(dir)
	// cleanupLoopのgoroutineが、後続のテストが書き換えるidleTimeout等の
	// パッケージ変数を読み続けてレースするのを防ぐため、必ず止める。
	t.Cleanup(st.stopCleanup)
	return st
}

func sessionDirExists(t *testing.T, st *sessionStore, sid string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(st.dataDir, sid))
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	t.Fatalf("stat session dir: %v", err)
	return false
}

// TestGetOrCreateEvictsLRUWhenOverMaxSessions は、maxSessionsを超えた状態で
// 新規セッションを作ると、最もlastAccessが古いセッションが追い出されることを
// 確認する。
func TestGetOrCreateEvictsLRUWhenOverMaxSessions(t *testing.T) {
	origMax := maxSessions
	maxSessions = 2
	t.Cleanup(func() { maxSessions = origMax })

	st := newTestSessionStore(t)

	if _, err := st.getOrCreate("a"); err != nil {
		t.Fatalf("getOrCreate(a) error: %v", err)
	}
	if _, err := st.getOrCreate("b"); err != nil {
		t.Fatalf("getOrCreate(b) error: %v", err)
	}

	// "a"を最もアイドルにするため、lastAccessを過去に巻き戻す。
	st.mu.Lock()
	st.sessions["a"].lastAccess = time.Now().Add(-time.Hour)
	st.mu.Unlock()

	// 上限(2)に達している状態で3件目を作ると、"a"が追い出されるはず。
	if _, err := st.getOrCreate("c"); err != nil {
		t.Fatalf("getOrCreate(c) error: %v", err)
	}

	if sessionDirExists(t, st, "a") {
		t.Error("session \"a\" (least recently used) should have been evicted")
	}
	if !sessionDirExists(t, st, "b") {
		t.Error("session \"b\" should still exist")
	}
	if !sessionDirExists(t, st, "c") {
		t.Error("session \"c\" should have been created")
	}
	if len(st.sessions) != 2 {
		t.Errorf("len(sessions) = %d, want 2", len(st.sessions))
	}
}

// TestCleanupEvictsIdleSessions は、idleTimeoutを超えて操作のないセッションが
// バックグラウンドの定期処理で削除されることを確認する。
func TestCleanupEvictsIdleSessions(t *testing.T) {
	origIdle, origInterval := idleTimeout, cleanupInterval
	idleTimeout = 20 * time.Millisecond
	cleanupInterval = 10 * time.Millisecond
	t.Cleanup(func() {
		idleTimeout, cleanupInterval = origIdle, origInterval
	})

	st := newTestSessionStore(t)
	if _, err := st.getOrCreate("idle-session"); err != nil {
		t.Fatalf("getOrCreate error: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !sessionDirExists(t, st, "idle-session") {
			return // 期待通り削除された
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("idle session was not cleaned up within the deadline")
}
