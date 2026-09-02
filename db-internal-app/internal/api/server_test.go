package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
)

func newCookieJar(t *testing.T, rawURL string) *cookiejar.Jar {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	if _, err := url.Parse(rawURL); err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	return jar
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	s := NewServer(dir)
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	srv := httptest.NewServer(WithCORS(mux))
	t.Cleanup(srv.Close)
	return srv
}

func postExec(t *testing.T, client *http.Client, url string, req execRequest) execResponse {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := client.Post(url+"/api/exec", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/exec: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	var out execResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func TestExecEndpointSetsSessionCookieAndPersists(t *testing.T) {
	srv := newTestServer(t)
	jar := newCookieJar(t, srv.URL)
	client := &http.Client{Jar: jar}

	res := postExec(t, client, srv.URL, execRequest{SQL: "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))"})
	if res.Error != "" {
		t.Fatalf("CREATE TABLE error: %s", res.Error)
	}

	res = postExec(t, client, srv.URL, execRequest{SQL: "INSERT INTO users VALUES (1, 'Alice')"})
	if res.Error != "" {
		t.Fatalf("INSERT error: %s", res.Error)
	}

	// 同じCookie(セッション)を使う2回目のリクエストで、1回目のCREATE TABLE/INSERTが
	// 見えることを確認する(セッションがリクエストをまたいで永続化していること)。
	res = postExec(t, client, srv.URL, execRequest{SQL: "SELECT * FROM users", Table: "users"})
	if res.Error != "" {
		t.Fatalf("SELECT error: %s", res.Error)
	}
	if len(res.Rows) != 1 {
		t.Fatalf("Rows = %d, want 1", len(res.Rows))
	}
	if res.Tree == nil {
		t.Fatal("Tree should be populated when table is specified")
	}
	root, ok := res.Tree.Pages[res.Tree.RootPageID]
	if !ok {
		t.Fatal("root page missing from tree response")
	}
	if !root.IsLeaf || len(root.Rows) != 1 {
		t.Errorf("root = %+v, want a leaf with 1 row", root)
	}
}

func TestExecEndpointReturnsParseErrorAsField(t *testing.T) {
	srv := newTestServer(t)
	jar := newCookieJar(t, srv.URL)
	client := &http.Client{Jar: jar}

	res := postExec(t, client, srv.URL, execRequest{SQL: "NOT VALID SQL"})
	if res.Error == "" {
		t.Fatal("expected a parse error to be reported in the Error field, not an HTTP failure")
	}
}

func TestResetClearsSessionData(t *testing.T) {
	srv := newTestServer(t)
	jar := newCookieJar(t, srv.URL)
	client := &http.Client{Jar: jar}

	postExec(t, client, srv.URL, execRequest{SQL: "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))"})
	postExec(t, client, srv.URL, execRequest{SQL: "INSERT INTO users VALUES (1, 'Alice')"})

	resp, err := client.Post(srv.URL+"/api/reset", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/reset: %v", err)
	}
	resp.Body.Close() //nolint:errcheck

	res := postExec(t, client, srv.URL, execRequest{SQL: "SELECT * FROM users"})
	if res.Error == "" {
		t.Fatal("expected an error selecting from a table that no longer exists after reset")
	}
}
