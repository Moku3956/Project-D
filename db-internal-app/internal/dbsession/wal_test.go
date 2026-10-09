package dbsession

import (
	"context"
	"testing"

	"github.com/Moku3956/Project-D/storage/wal"
)

func TestWalRecordsReflectsInsertUpdateDelete(t *testing.T) {
	s := setupSession(t)

	if _, err := s.Exec(context.Background(), "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))"); err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}
	if _, err := s.Exec(context.Background(), "INSERT INTO users VALUES (1, 'Alice')"); err != nil {
		t.Fatalf("INSERT error: %v", err)
	}
	if _, err := s.Exec(context.Background(), "UPDATE users SET name = 'Alicia' WHERE id = 1"); err != nil {
		t.Fatalf("UPDATE error: %v", err)
	}
	if _, err := s.Exec(context.Background(), "DELETE FROM users WHERE id = 1"); err != nil {
		t.Fatalf("DELETE error: %v", err)
	}

	records, err := s.WalRecords()
	if err != nil {
		t.Fatalf("WalRecords error: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("expected some WAL records")
	}

	var insert, del *WalRecord
	for i := range records {
		switch records[i].Op {
		case "INSERT":
			if insert == nil {
				insert = &records[i]
			}
		case "DELETE":
			del = &records[i]
		}
	}
	if insert == nil {
		t.Fatal("no INSERT record found")
	}
	if insert.ChangeKind != "added" || insert.Changed == nil {
		t.Errorf("INSERT record: ChangeKind = %q, Changed = %v, want added/non-nil", insert.ChangeKind, insert.Changed)
	}
	if del == nil {
		t.Fatal("no DELETE record found")
	}
	if del.ChangeKind != "removed" || del.Changed == nil {
		t.Errorf("DELETE record: ChangeKind = %q, Changed = %v, want removed/non-nil", del.ChangeKind, del.Changed)
	}

	// COMMITレコードも(TxnIDごとに)含まれていること。
	foundCommit := false
	for _, r := range records {
		if r.Op == "COMMIT" {
			foundCommit = true
			break
		}
	}
	if !foundCommit {
		t.Error("expected at least one COMMIT record")
	}
}

// txn.Manager.RollbackはOpAbortをwm.bufに積むだけでFlushしない
// (storage/wal/docs/spec.md参照)。WalRecordsが読む前に自分でFlushすることで、
// 次のコミットを待たずにABORTがすぐ見えることを確認する。
func TestWalRecordsFlushesBeforeReading(t *testing.T) {
	s := setupSession(t)

	if _, err := s.wm.Append(&wal.LogRecord{TxnID: 99, Op: wal.OpAbort}); err != nil {
		t.Fatalf("Append error: %v", err)
	}

	records, err := s.WalRecords()
	if err != nil {
		t.Fatalf("WalRecords error: %v", err)
	}

	found := false
	for _, r := range records {
		if r.TxnID == 99 && r.Op == "ABORT" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected the unflushed ABORT record to be visible without a later commit")
	}
}
