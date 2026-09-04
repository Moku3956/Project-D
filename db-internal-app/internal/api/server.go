package api

import (
	"encoding/json"
	"net/http"

	"github.com/Moku3956/Project-D/storage/btree"
	"github.com/Moku3956/Project-D/types"
)

const sessionCookieName = "db_internal_sid"

// Server はdb-internal-appのHTTP APIサーバー。
type Server struct {
	sessions *sessionStore
}

// NewServer はdataDir配下にセッションごとのディレクトリを作るサーバーを返す。
func NewServer(dataDir string) *Server {
	return &Server{sessions: newSessionStore(dataDir)}
}

// RegisterRoutes はハンドラをmuxに登録する。
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /api/tables", s.handleTables)
	mux.HandleFunc("POST /api/exec", s.handleExec)
	mux.HandleFunc("POST /api/reset", s.handleReset)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type execRequest struct {
	SQL   string `json:"sql"`
	Table string `json:"table,omitempty"`
}

type execResponse struct {
	Columns      []string          `json:"columns,omitempty"`
	Rows         [][]any           `json:"rows,omitempty"`
	AffectedRows int               `json:"affectedRows"`
	Tree         *treeSnapshotJSON `json:"tree,omitempty"`
	Error        string            `json:"error,omitempty"`
}

func (s *Server) handleExec(w http.ResponseWriter, r *http.Request) {
	sid, err := s.sessionID(w, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sess, err := s.sessions.getOrCreate(sid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var req execRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	result, err := sess.Exec(req.SQL)
	if err != nil {
		writeJSON(w, http.StatusOK, execResponse{Error: err.Error()})
		return
	}

	resp := execResponse{AffectedRows: result.AffectedRows}
	if result.Schema != nil {
		cols := make([]string, len(result.Schema.Columns))
		for i, c := range result.Schema.Columns {
			cols[i] = c.Name
		}
		resp.Columns = cols
		resp.Rows = marshalRows(result.Rows)
	}

	if req.Table != "" {
		snap, err := sess.DumpTree(req.Table)
		if err != nil {
			writeJSON(w, http.StatusOK, execResponse{
				Columns: resp.Columns, Rows: resp.Rows, AffectedRows: resp.AffectedRows,
				Error: "query succeeded but DumpTree failed: " + err.Error(),
			})
			return
		}
		resp.Tree = marshalTreeSnapshot(snap)
	}

	writeJSON(w, http.StatusOK, resp)
}

type tableInfoJSON struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

// handleTables はセッション内の全テーブル(名前・カラム一覧)を返す
// (テーブル切り替えタブUI用)。
func (s *Server) handleTables(w http.ResponseWriter, r *http.Request) {
	sid, err := s.sessionID(w, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sess, err := s.sessions.getOrCreate(sid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tables, err := sess.Tables()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]tableInfoJSON, len(tables))
	for i, t := range tables {
		out[i] = tableInfoJSON{Name: t.Name, Columns: t.Columns}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	sid, err := s.sessionID(w, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.sessions.reset(sid); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
}

// sessionID はリクエストのCookieからセッションIDを読み、なければ新規発行して
// レスポンスにCookieをセットする。
func (s *Server) sessionID(w http.ResponseWriter, r *http.Request) (string, error) {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		return c.Value, nil
	}
	sid, err := newSessionID()
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return sid, nil
}

type treeSnapshotJSON struct {
	RootPageID uint32                      `json:"rootPageId"`
	Pages      map[uint32]pageSnapshotJSON `json:"pages"`
}

type pageSnapshotJSON struct {
	PageID         uint32   `json:"pageId"`
	IsLeaf         bool     `json:"isLeaf"`
	Keys           []string `json:"keys,omitempty"`
	ChildPageIDs   []uint32 `json:"childPageIds,omitempty"`
	RightmostChild uint32   `json:"rightmostChild,omitempty"`
	Rows           [][]any  `json:"rows,omitempty"`
	RowTables      []string `json:"rowTables,omitempty"`
	NextLeafID     uint32   `json:"nextLeafId,omitempty"`
}

func marshalTreeSnapshot(snap *btree.TreeSnapshot) *treeSnapshotJSON {
	pages := make(map[uint32]pageSnapshotJSON, len(snap.Pages))
	for id, ps := range snap.Pages {
		pages[id] = pageSnapshotJSON{
			PageID:         ps.PageID,
			IsLeaf:         ps.IsLeaf,
			Keys:           ps.Keys,
			ChildPageIDs:   ps.ChildPageIDs,
			RightmostChild: ps.RightmostChild,
			Rows:           marshalRows(ps.Rows),
			RowTables:      ps.RowTables,
			NextLeafID:     ps.NextLeafID,
		}
	}
	return &treeSnapshotJSON{RootPageID: snap.RootPageID, Pages: pages}
}

func marshalRows(rows []types.Row) [][]any {
	if len(rows) == 0 {
		return nil
	}
	out := make([][]any, len(rows))
	for i, row := range rows {
		vals := make([]any, len(row.Values))
		for j, v := range row.Values {
			vals[j] = marshalValue(v)
		}
		out[i] = vals
	}
	return out
}

func marshalValue(v types.Value) any {
	switch val := v.(type) {
	case types.IntValue:
		return val.V
	case types.StringValue:
		return val.V
	case types.BoolValue:
		return val.V
	case types.NullValue:
		return nil
	default:
		return nil
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// WithCORS はローカル開発用のCORSミドルウェア。Cookie(credentials)を使うため
// Access-Control-Allow-Origin にワイルドカードは使えず、リクエスト元を
// そのまま反映する。
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
