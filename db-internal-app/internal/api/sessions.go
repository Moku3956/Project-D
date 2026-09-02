// Package api はdb-internal-appのHTTP APIを提供する。
package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Moku3956/Project-D/db-internal-app/internal/dbsession"
)

// sessionStore はCookieのセッションIDと*dbsession.Sessionを対応付ける
// in-memoryマップ。セッションごとに専用のディレクトリを遅延作成する
// (db-internal-app/docs/spec.md「セッションの永続性」参照)。
//
// アイドルセッションの掃除・同時セッション数の上限は未実装(spec.mdの
// 「未定事項」参照)。将来公開する際は必須だが、今回はB+Tree可視化の
// APIを動かすことを優先し、スコープ外にしている。
type sessionStore struct {
	mu       sync.Mutex
	dataDir  string
	sessions map[string]*dbsession.Session
}

func newSessionStore(dataDir string) *sessionStore {
	return &sessionStore{
		dataDir:  dataDir,
		sessions: make(map[string]*dbsession.Session),
	}
}

// getOrCreate はsidに対応するセッションを返す。存在しなければ専用ディレクトリを
// 作って新規に開く。
func (st *sessionStore) getOrCreate(sid string) (*dbsession.Session, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if s, ok := st.sessions[sid]; ok {
		return s, nil
	}
	dir := filepath.Join(st.dataDir, sid)
	s, err := dbsession.Open(dir)
	if err != nil {
		return nil, fmt.Errorf("open session %s: %w", sid, err)
	}
	st.sessions[sid] = s
	return s, nil
}

// reset はsidのセッションを閉じてディスク上のディレクトリごと削除し、
// 次回アクセス時に空の状態から作り直せるようにする(「リセット」ボタン用)。
func (st *sessionStore) reset(sid string) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if s, ok := st.sessions[sid]; ok {
		if err := s.Close(); err != nil {
			return err
		}
		delete(st.sessions, sid)
	}
	dir := filepath.Join(st.dataDir, sid)
	return os.RemoveAll(dir)
}

// newSessionID はセッションCookie用のランダムなIDを生成する。
func newSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
