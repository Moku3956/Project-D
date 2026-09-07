// Package api はdb-internal-appのHTTP APIを提供する。
package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Moku3956/Project-D/db-internal-app/internal/dbsession"
)

// maxSessions はサーバー全体で同時に保持できるセッション数の上限。超えた状態で
// 新規セッションが来た場合、最もアイドルなセッションを追い出す(spec.md
// 「公開デモとして必要なリソース制限」参照)。テストで小さい値に差し替えられる
// よう定数ではなく変数にしている。
var maxSessions = 500

// idleTimeout はこの時間操作がないセッションを、バックグラウンドの定期処理で
// ディスク上のディレクトリごと削除する(spec.md「アイドルセッションの掃除」参照)。
var idleTimeout = 30 * time.Minute

// cleanupInterval はアイドルセッションを探す定期処理の実行間隔。
var cleanupInterval = time.Minute

// sessionEntry はセッション本体と、追い出し判定に使う最終アクセス時刻の組。
type sessionEntry struct {
	sess       *dbsession.Session
	lastAccess time.Time
}

// sessionStore はCookieのセッションIDと*dbsession.Sessionを対応付ける
// in-memoryマップ。セッションごとに専用のディレクトリを遅延作成する
// (db-internal-app/docs/spec.md「セッションの永続性」参照)。
type sessionStore struct {
	mu       sync.Mutex
	dataDir  string
	sessions map[string]*sessionEntry
	stop     chan struct{}
	done     chan struct{}
}

func newSessionStore(dataDir string) *sessionStore {
	st := &sessionStore{
		dataDir:  dataDir,
		sessions: make(map[string]*sessionEntry),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	go st.cleanupLoop()
	return st
}

// stopCleanup はバックグラウンドのcleanupLoopを止め、goroutineが実際に終了する
// まで待つ。本番のサーバープロセスは生存期間中ずっと動かし続けるため呼ばない
// 想定だが、テストではgoroutineがパッケージ変数(idleTimeout等)を読み続けて
// 後続のテストとレースしないよう、必ず呼ぶこと。
func (st *sessionStore) stopCleanup() {
	close(st.stop)
	<-st.done
}

// getOrCreate はsidに対応するセッションを返す。存在しなければ専用ディレクトリを
// 作って新規に開く。既存セッション数がmaxSessionsに達している場合は、新規作成の
// 前に最もアイドルなセッションを1つ追い出す。
func (st *sessionStore) getOrCreate(sid string) (*dbsession.Session, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if e, ok := st.sessions[sid]; ok {
		e.lastAccess = time.Now()
		return e.sess, nil
	}

	if len(st.sessions) >= maxSessions {
		if err := st.evictLRULocked(); err != nil {
			return nil, err
		}
	}

	dir := filepath.Join(st.dataDir, sid)
	s, err := dbsession.Open(dir)
	if err != nil {
		return nil, fmt.Errorf("open session %s: %w", sid, err)
	}
	st.sessions[sid] = &sessionEntry{sess: s, lastAccess: time.Now()}
	return s, nil
}

// reset はsidのセッションを閉じてディスク上のディレクトリごと削除し、
// 次回アクセス時に空の状態から作り直せるようにする(「リセット」ボタン用)。
func (st *sessionStore) reset(sid string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.evictLocked(sid)
}

// evictLocked はsidのセッションを閉じてディレクトリごと削除する。存在しなくても
// (ディレクトリの削除だけ試みて)エラーにはしない。呼び出し元でstd.muをロック
// 済みであること。
func (st *sessionStore) evictLocked(sid string) error {
	if e, ok := st.sessions[sid]; ok {
		if err := e.sess.Close(); err != nil {
			return err
		}
		delete(st.sessions, sid)
	}
	dir := filepath.Join(st.dataDir, sid)
	return os.RemoveAll(dir)
}

// evictLRULocked は最もlastAccessが古いセッションを1つ追い出す。呼び出し元で
// st.muをロック済みであること。
func (st *sessionStore) evictLRULocked() error {
	var oldestSID string
	var oldestAccess time.Time
	for sid, e := range st.sessions {
		if oldestSID == "" || e.lastAccess.Before(oldestAccess) {
			oldestSID = sid
			oldestAccess = e.lastAccess
		}
	}
	if oldestSID == "" {
		return nil
	}
	return st.evictLocked(oldestSID)
}

// cleanupLoop はcleanupInterval間隔で全セッションを走査し、idleTimeoutを超えて
// 操作のないセッションを追い出す。st.stopが閉じられるまで動き続ける(本番の
// サーバープロセスではstopCleanupを呼ばないため、プロセスの生存期間だけ動く)。
func (st *sessionStore) cleanupLoop() {
	defer close(st.done)
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			st.evictIdle()
		case <-st.stop:
			return
		}
	}
}

// evictIdle はidleTimeoutを超えてアクセスのないセッションをすべて追い出す。
func (st *sessionStore) evictIdle() {
	st.mu.Lock()
	defer st.mu.Unlock()

	now := time.Now()
	for sid, e := range st.sessions {
		if now.Sub(e.lastAccess) > idleTimeout {
			_ = st.evictLocked(sid) //nolint:errcheck
		}
	}
}

// newSessionID はセッションCookie用のランダムなIDを生成する。
func newSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
