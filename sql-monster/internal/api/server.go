// Package api はsql-monsterのHTTP APIを提供する。フロントエンド(React)からの
// 呼び出しを受け、internal/gameの対戦ロジックに橋渡しする層。
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/Moku3956/Project-D/sql-monster/internal/game"
	"github.com/Moku3956/Project-D/sql-monster/internal/sqlitedb"
	"github.com/Moku3956/Project-D/types"
)

// Phase は1ターンの4フェーズ(spec.md「1ターンの流れ」)を表す。
type Phase int

const (
	PhaseAttackAnalysis  Phase = iota + 1 // ①攻撃用分析
	PhaseAttackExec                       // ②攻撃
	PhaseDefenseAnalysis                  // ③防御用分析
	PhaseDefenseExec                      // ④防御
)

func (p Phase) String() string {
	switch p {
	case PhaseAttackAnalysis:
		return "Attack Analysis"
	case PhaseAttackExec:
		return "Attack Exec"
	case PhaseDefenseAnalysis:
		return "Defense Analysis"
	case PhaseDefenseExec:
		return "Defense Exec"
	}
	return "Unknown"
}

// isAnalysis は分析フェーズ(SELECTを何度も撃てて、手動で次に進むフェーズ)かを返す。
func (p Phase) isAnalysis() bool {
	return p == PhaseAttackAnalysis || p == PhaseDefenseAnalysis
}

// session は進行中のバトル1件分。サーバーのメモリ上にのみ置く(プロトタイプのため
// 再起動で消えることは許容。docs/frontend_architecture.md参照)。
type session struct {
	id      string
	monster game.Monster
	battle  *game.Battle
	phase   Phase
	turn    int
	logs    []string
	over    bool
	won     bool
}

func (s *session) log(format string, args ...interface{}) {
	s.logs = append(s.logs, fmt.Sprintf(format, args...))
	if len(s.logs) > 50 {
		s.logs = s.logs[len(s.logs)-50:]
	}
}

// Server はHTTPハンドラと進行中バトルの保持を担う。
type Server struct {
	db *sqlitedb.DB

	mu       sync.Mutex
	sessions map[string]*session
	seq      int
}

func NewServer(db *sqlitedb.DB) *Server {
	return &Server{db: db, sessions: make(map[string]*session)}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/monsters", s.handleMonsters)
	mux.HandleFunc("POST /api/battles", s.handleCreateBattle)
	mux.HandleFunc("GET /api/battles/{id}", s.handleGetBattle)
	mux.HandleFunc("POST /api/battles/{id}/query", s.handleQuery)
	mux.HandleFunc("POST /api/battles/{id}/advance", s.handleAdvance)
	mux.HandleFunc("POST /api/battles/{id}/quit", s.handleQuit)
	mux.HandleFunc("POST /api/battles/{id}/restart", s.handleRestart)
}

// ---- ハンドラ ----

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleMonsters はホーム画面のパス用に、進行状態つきのモンスター一覧を返す。
func (s *Server) handleMonsters(w http.ResponseWriter, r *http.Request) {
	cleared, err := game.ClearedIDs(s.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	all := game.Monsters()
	views := make([]monsterView, 0, len(all))
	currentAssigned := false
	for _, m := range all {
		state := "locked"
		switch {
		case cleared[m.ID]:
			state = "cleared"
		case !currentAssigned:
			state = "current"
			currentAssigned = true
		}
		views = append(views, newMonsterView(m, state))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"monsters": views})
}

type createBattleRequest struct {
	MonsterID int64 `json:"monster_id"`
}

func (s *Server) handleCreateBattle(w http.ResponseWriter, r *http.Request) {
	var req createBattleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "リクエストの形式が正しくありません")
		return
	}
	monster, ok := game.FindMonster(req.MonsterID)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("モンスター %d が見つかりません", req.MonsterID))
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sess, err := s.startSession(monster)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	view, err := s.viewOf(sess)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (s *Server) handleGetBattle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[r.PathValue("id")]
	if !ok {
		writeError(w, http.StatusNotFound, "バトルが見つかりません")
		return
	}
	view, err := s.viewOf(sess)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

type queryRequest struct {
	SQL string `json:"sql"`
}

// handleQuery は現在のフェーズに応じてSQLを実行する。分析フェーズはそのフェーズに
// 留まり、実行フェーズ(②④)は実行と同時に次のフェーズへ進む。
func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "リクエストの形式が正しくありません")
		return
	}
	if req.SQL == "" {
		writeError(w, http.StatusBadRequest, "sql フィールドが必要です")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[r.PathValue("id")]
	if !ok {
		writeError(w, http.StatusNotFound, "バトルが見つかりません")
		return
	}
	if sess.over {
		writeError(w, http.StatusConflict, "このバトルは既に決着しています")
		return
	}

	result, err := s.runPhaseQuery(sess, req.SQL)
	if err != nil {
		// SQLのエラーはゲーム内の出来事として扱い、ログに載せて200で返す
		sess.log(">_ ERROR: %s", err.Error())
		view, verr := s.viewOf(sess)
		if verr != nil {
			writeError(w, http.StatusInternalServerError, verr.Error())
			return
		}
		writeJSON(w, http.StatusOK, queryResponse{Error: err.Error(), Battle: view})
		return
	}

	resp := queryResponse{}
	if result != nil {
		resp.Columns, resp.Rows = marshalRows(result)
		resp.Scan = scanName(result.Scan)
	}
	view, err := s.viewOf(sess)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Battle = view
	writeJSON(w, http.StatusOK, resp)
}

// handleAdvance は分析フェーズから次のフェーズへ手動で進める。
func (s *Server) handleAdvance(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[r.PathValue("id")]
	if !ok {
		writeError(w, http.StatusNotFound, "バトルが見つかりません")
		return
	}
	if sess.over {
		writeError(w, http.StatusConflict, "このバトルは既に決着しています")
		return
	}
	if !sess.phase.isAnalysis() {
		writeError(w, http.StatusConflict, "このフェーズはクエリの実行で自動的に進みます")
		return
	}

	if err := s.advance(sess); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	view, err := s.viewOf(sess)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// handleQuit はバトルを放棄する。その対戦は敗北扱いになる
// (docs/battle_screen_ui.md「設定パネル(バトルメニュー)」)。
func (s *Server) handleQuit(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[r.PathValue("id")]
	if !ok {
		writeError(w, http.StatusNotFound, "バトルが見つかりません")
		return
	}
	sess.over = true
	sess.won = false
	sess.log(">_ Battle abandoned. Recorded as a defeat.")

	view, err := s.viewOf(sess)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// handleRestart は同じモンスターでバトルを組み直す。ペナルティはない。
func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.sessions[r.PathValue("id")]
	if !ok {
		writeError(w, http.StatusNotFound, "バトルが見つかりません")
		return
	}
	delete(s.sessions, old.id)

	sess, err := s.startSession(old.monster)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	view, err := s.viewOf(sess)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// ---- 内部処理 ----

// startSession はHP・リソースをリセットして新しいバトルを開始する。呼び出し側でロック済みであること。
func (s *Server) startSession(m game.Monster) (*session, error) {
	if err := game.ResetHP(s.db, m); err != nil {
		return nil, err
	}
	battle, err := game.NewBattle(s.db, game.PlayerID, m.ID, game.ResourcesFor(m.Level))
	if err != nil {
		return nil, err
	}

	s.seq++
	sess := &session{
		id:      fmt.Sprintf("b%d", s.seq),
		monster: m,
		battle:  battle,
		phase:   PhaseAttackAnalysis,
		turn:    1,
	}
	sess.log(">_ Encounter started: %s (Lv%d)", m.Name, m.Level)
	s.sessions[sess.id] = sess
	return sess, nil
}

// runPhaseQuery は現在のフェーズに対応する処理を実行する。呼び出し側でロック済みであること。
func (s *Server) runPhaseQuery(sess *session, sql string) (*sqlitedb.Result, error) {
	switch sess.phase {
	case PhaseAttackAnalysis:
		result, err := sess.battle.AnalyzeWeakness(sql)
		if err != nil {
			return nil, err
		}
		sess.log(">_ SELECT returned %d row(s). ANALYSIS_AP -%d", len(result.Rows), len(result.Rows))
		return result, nil

	case PhaseAttackExec:
		dealt, ok, err := sess.battle.Attack(sql)
		if err != nil {
			return nil, err
		}
		if !ok {
			sess.log(">_ ROLLBACK: measured damage exceeds CRUD_ATTACK_AP. Attack fizzled.")
		} else {
			sess.log(">_ COMMIT: dealt %d damage. CRUD_ATTACK_AP -%d", dealt, dealt)
		}
		if err := s.advance(sess); err != nil {
			return nil, err
		}
		return nil, nil

	case PhaseDefenseAnalysis:
		result, err := sess.battle.AnalyzeDefense(sql)
		if err != nil {
			return nil, err
		}
		sess.log(">_ SELECT returned %d row(s). ANALYSIS_AP -%d", len(result.Rows), len(result.Rows))
		return result, nil

	case PhaseDefenseExec:
		result, blockRate, damage, err := sess.battle.Defend(sql)
		if err != nil {
			return nil, err
		}
		if blockRate == 0 {
			sess.log(">_ Block failed. Took %d damage.", damage)
		} else {
			sess.log(">_ Block rate %.0f%%. Took %d damage.", blockRate*100, damage)
		}
		if err := s.advance(sess); err != nil {
			return result, err
		}
		return result, nil
	}
	return nil, fmt.Errorf("不正なフェーズです")
}

// advance は次のフェーズへ進める。④の次はターンを終えて①へ戻る。
func (s *Server) advance(sess *session) error {
	if sess.phase == PhaseDefenseExec {
		if err := s.checkOver(sess); err != nil {
			return err
		}
		if sess.over {
			return nil
		}
		if err := sess.battle.NextTurn(); err != nil {
			return err
		}
		sess.turn++
		sess.phase = PhaseAttackAnalysis
		sess.log(">_ Turn %d start. Resources restored.", sess.turn)
		return nil
	}

	sess.phase++
	if err := s.checkOver(sess); err != nil {
		return err
	}
	return nil
}

// checkOver は決着がついていればセッションに反映し、勝利なら進行状況を保存する。
func (s *Server) checkOver(sess *session) error {
	if sess.over {
		return nil
	}
	over, won, err := sess.battle.IsOver()
	if err != nil {
		return err
	}
	if !over {
		return nil
	}
	sess.over = true
	sess.won = won
	if won {
		sess.log(">_ VICTORY: %s defeated.", sess.monster.Name)
		return game.MarkCleared(s.db, sess.monster.ID)
	}
	sess.log(">_ DEFEAT: player HP reached 0.")
	return nil
}

// viewOf は現在のバトル状態をレスポンス用の形に変換する。呼び出し側でロック済みであること。
func (s *Server) viewOf(sess *session) (battleView, error) {
	playerHP, err := sess.battle.PlayerHP()
	if err != nil {
		return battleView{}, err
	}
	monsterHP, err := sess.battle.MonsterHP()
	if err != nil {
		return battleView{}, err
	}

	res := sess.battle.Resources()
	max := sess.battle.MaxResources()

	state := "current"
	if sess.over {
		if sess.won {
			state = "cleared"
		} else {
			state = "current"
		}
	}

	return battleView{
		ID:          sess.id,
		Monster:     newMonsterView(sess.monster, state),
		Phase:       int(sess.phase),
		PhaseName:   sess.phase.String(),
		Turn:        sess.turn,
		PlayerHP:    playerHP,
		PlayerMaxHP: game.PlayerMaxHP,
		MonsterHP:   monsterHP,
		Resources: resourceView{
			Analysis:         res.Analysis,
			AnalysisMax:      max.Analysis,
			AttackDefense:    res.AttackDefense,
			AttackDefenseMax: max.AttackDefense,
		},
		Logs: sess.logs,
		Over: sess.over,
		Won:  sess.won,
	}, nil
}

// ---- レスポンスの型とヘルパー ----

type monsterView struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Level  int64  `json:"level"`
	MaxHP  int64  `json:"max_hp"`
	IsBoss bool   `json:"is_boss"`
	State  string `json:"state"` // cleared / current / locked
}

func newMonsterView(m game.Monster, state string) monsterView {
	return monsterView{
		ID:     m.ID,
		Name:   m.Name,
		Level:  m.Level,
		MaxHP:  m.HP,
		IsBoss: m.IsBoss(),
		State:  state,
	}
}

type resourceView struct {
	Analysis         int64 `json:"analysis"`
	AnalysisMax      int64 `json:"analysis_max"`
	AttackDefense    int64 `json:"attack_defense"`
	AttackDefenseMax int64 `json:"attack_defense_max"`
}

type battleView struct {
	ID          string       `json:"id"`
	Monster     monsterView  `json:"monster"`
	Phase       int          `json:"phase"`
	PhaseName   string       `json:"phase_name"`
	Turn        int          `json:"turn"`
	PlayerHP    int64        `json:"player_hp"`
	PlayerMaxHP int64        `json:"player_max_hp"`
	MonsterHP   int64        `json:"monster_hp"`
	Resources   resourceView `json:"resources"`
	Logs        []string     `json:"logs"`
	Over        bool         `json:"over"`
	Won         bool         `json:"won"`
}

type queryResponse struct {
	Columns []string        `json:"columns,omitempty"`
	Rows    [][]interface{} `json:"rows,omitempty"`
	Scan    string          `json:"scan,omitempty"`
	Error   string          `json:"error,omitempty"`
	Battle  battleView      `json:"battle"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// scanName はプランが選んだスキャン戦略を文字列にする。
// 精密なクエリ(IndexScan)かどうかがクリティカル判定の材料になる(spec.md参照)。
func scanName(k sqlitedb.ScanKind) string {
	switch k {
	case sqlitedb.ScanIndex:
		return "index"
	case sqlitedb.ScanSequential:
		return "sequential"
	}
	return "none"
}

func marshalRows(result *sqlitedb.Result) ([]string, [][]interface{}) {
	if result.Schema == nil || len(result.Rows) == 0 {
		return nil, nil
	}
	cols := make([]string, len(result.Schema.Columns))
	for i, c := range result.Schema.Columns {
		cols[i] = c.Name
	}
	rows := make([][]interface{}, len(result.Rows))
	for i, row := range result.Rows {
		vals := make([]interface{}, len(row.Values))
		for j, v := range row.Values {
			vals[j] = marshalValue(v)
		}
		rows[i] = vals
	}
	return cols, rows
}

func marshalValue(v types.Value) interface{} {
	switch val := v.(type) {
	case types.IntValue:
		return val.V
	case types.StringValue:
		return val.V
	case types.BoolValue:
		return val.V
	case types.NullValue:
		return nil
	}
	return v
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

// WithCORS は開発中のViteサーバー(別ポート)からの呼び出しを許可する。
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
