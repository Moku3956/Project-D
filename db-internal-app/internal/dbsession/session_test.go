package dbsession

import (
	"fmt"
	"testing"
)

func setupSession(t *testing.T) *Session {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestExecCreateTableAndInsert(t *testing.T) {
	s := setupSession(t)

	if _, err := s.Exec("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))"); err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}
	if _, err := s.Exec("INSERT INTO users VALUES (1, 'Alice')"); err != nil {
		t.Fatalf("INSERT error: %v", err)
	}

	res, err := s.Exec("SELECT * FROM users")
	if err != nil {
		t.Fatalf("SELECT error: %v", err)
	}
	if len(res.Rows) != 1 {
		t.Fatalf("Rows = %d, want 1", len(res.Rows))
	}
}

func TestDumpTreeReflectsInserts(t *testing.T) {
	s := setupSession(t)

	if _, err := s.Exec("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))"); err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}
	for i := 1; i <= 3; i++ {
		sql := fmt.Sprintf("INSERT INTO users VALUES (%d, 'user-%d')", i, i)
		if _, err := s.Exec(sql); err != nil {
			t.Fatalf("INSERT(%d) error: %v", i, err)
		}
	}

	snap, err := s.DumpTree("users")
	if err != nil {
		t.Fatalf("DumpTree error: %v", err)
	}
	root, ok := snap.Pages[snap.RootPageID]
	if !ok {
		t.Fatal("root page missing from snapshot")
	}
	if !root.IsLeaf {
		t.Fatal("root should still be a single leaf for 3 rows")
	}
	if len(root.Rows) != 3 {
		t.Errorf("Rows = %d, want 3", len(root.Rows))
	}
}

func TestDumpTreeAfterSplitViaExec(t *testing.T) {
	s := setupSession(t)

	if _, err := s.Exec("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))"); err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}
	const n = 300
	for i := 1; i <= n; i++ {
		sql := fmt.Sprintf("INSERT INTO users VALUES (%d, 'user-%d')", i, i)
		if _, err := s.Exec(sql); err != nil {
			t.Fatalf("INSERT(%d) error: %v", i, err)
		}
	}

	snap, err := s.DumpTree("users")
	if err != nil {
		t.Fatalf("DumpTree error: %v", err)
	}
	root := snap.Pages[snap.RootPageID]
	if root.IsLeaf {
		t.Fatal("root should have become internal after enough inserts to split")
	}

	total := 0
	for _, ps := range snap.Pages {
		if ps.IsLeaf {
			total += len(ps.Rows)
		}
	}
	if total != n {
		t.Errorf("total rows across leaves = %d, want %d", total, n)
	}
}

func TestDumpTreeUnknownTableErrors(t *testing.T) {
	s := setupSession(t)
	if _, err := s.DumpTree("nope"); err == nil {
		t.Fatal("DumpTree on an unopened table should error")
	}
}
