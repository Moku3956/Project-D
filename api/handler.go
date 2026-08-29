package api

import (
	"encoding/json"
	"net/http"

	"github.com/Moku3956/Project-D/executor"
	"github.com/Moku3956/Project-D/sql/parser"
	"github.com/Moku3956/Project-D/sql/planner"
	"github.com/Moku3956/Project-D/types"
)

type queryRequest struct {
	SQL string `json:"sql"`
}

type queryResponse struct {
	Columns      []string        `json:"columns,omitempty"`
	Rows         [][]interface{} `json:"rows,omitempty"`
	AffectedRows int             `json:"affected_rows,omitempty"`
	Error        string          `json:"error,omitempty"`
}

type Handler struct {
	planner *planner.Planner
	engine  *executor.Engine
}

func NewHandler(pl *planner.Planner, eng *executor.Engine) *Handler {
	return &Handler{planner: pl, engine: eng}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /query", h.handleQuery)
	mux.HandleFunc("GET /health", h.handleHealth)
}

func (h *Handler) handleQuery(w http.ResponseWriter, r *http.Request) {
	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "リクエストの形式が正しくありません")
		return
	}
	if req.SQL == "" {
		writeError(w, http.StatusBadRequest, "sql フィールドが必要です")
		return
	}

	stmt, err := parser.Parse(req.SQL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	node, err := h.planner.Plan(stmt)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.engine.Execute(node)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := queryResponse{AffectedRows: result.AffectedRows}
	if result.Schema != nil && len(result.Rows) > 0 {
		cols := make([]string, len(result.Schema.Columns))
		for i, col := range result.Schema.Columns {
			cols[i] = col.Name
		}
		resp.Columns = cols

		rows := make([][]interface{}, len(result.Rows))
		for i, row := range result.Rows {
			vals := make([]interface{}, len(row.Values))
			for j, v := range row.Values {
				vals[j] = marshalValue(v)
			}
			rows[i] = vals
		}
		resp.Rows = rows
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, queryResponse{Error: msg})
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
