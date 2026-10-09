package wal

import (
	"os"
	"testing"
)

func tempWAL(t *testing.T) (string, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "wal_test_*.wal")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(f.Name()); err != nil {
		t.Fatal(err)
	}
	return f.Name(), func() {
		if err := os.Remove(f.Name()); err != nil && !os.IsNotExist(err) {
			t.Errorf("cleanup: %v", err)
		}
	}
}

// ---- 正常系 ----

func TestAppendIncrementLSN(t *testing.T) {
	path, cleanup := tempWAL(t)
	defer cleanup()

	wm, err := NewWALManager(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := wm.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	for i := uint64(0); i < 3; i++ {
		lsn, err := wm.Append(&LogRecord{TxnID: 1, Op: OpInsert})
		if err != nil {
			t.Fatal(err)
		}
		if lsn != i {
			t.Errorf("LSN = %d, want %d", lsn, i)
		}
	}
}

func TestFlushAndReadAll(t *testing.T) {
	path, cleanup := tempWAL(t)
	defer cleanup()

	wm, err := NewWALManager(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := wm.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	if _, err := wm.Append(&LogRecord{TxnID: 1, Op: OpInsert}); err != nil {
		t.Fatal(err)
	}
	if _, err := wm.Append(&LogRecord{TxnID: 1, Op: OpCommit}); err != nil {
		t.Fatal(err)
	}
	if err := wm.Flush(); err != nil {
		t.Fatal(err)
	}

	records, err := wm.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Errorf("件数 = %d, want 2", len(records))
	}
	if records[0].Op != OpInsert {
		t.Errorf("records[0].Op = %v, want OpInsert", records[0].Op)
	}
	if records[1].Op != OpCommit {
		t.Errorf("records[1].Op = %v, want OpCommit", records[1].Op)
	}
}

func TestReopenRestoresNextLSN(t *testing.T) {
	path, cleanup := tempWAL(t)
	defer cleanup()

	wm, err := NewWALManager(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wm.Append(&LogRecord{TxnID: 1, Op: OpInsert}); err != nil {
		t.Fatal(err)
	}
	if _, err := wm.Append(&LogRecord{TxnID: 1, Op: OpCommit}); err != nil {
		t.Fatal(err)
	}
	if err := wm.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := wm.Close(); err != nil {
		t.Fatal(err)
	}

	wm2, err := NewWALManager(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := wm2.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	lsn, err := wm2.Append(&LogRecord{TxnID: 2, Op: OpInsert})
	if err != nil {
		t.Fatal(err)
	}
	if lsn != 2 {
		t.Errorf("再オープン後のLSN = %d, want 2", lsn)
	}
}

func TestRedoDataRoundtrip(t *testing.T) {
	path, cleanup := tempWAL(t)
	defer cleanup()

	wm, err := NewWALManager(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := wm.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	data := []byte{0x01, 0x02, 0x03, 0x04}
	if _, err := wm.Append(&LogRecord{TxnID: 1, Op: OpInsert, RedoData: data}); err != nil {
		t.Fatal(err)
	}
	if err := wm.Flush(); err != nil {
		t.Fatal(err)
	}

	records, err := wm.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("件数 = %d, want 1", len(records))
	}
	if string(records[0].RedoData) != string(data) {
		t.Errorf("RedoData = %v, want %v", records[0].RedoData, data)
	}
}

// ---- 異常系 ----

func TestReadAllBeforeFlush(t *testing.T) {
	path, cleanup := tempWAL(t)
	defer cleanup()

	wm, err := NewWALManager(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := wm.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	if _, err := wm.Append(&LogRecord{TxnID: 1, Op: OpInsert}); err != nil {
		t.Fatal(err)
	}

	records, err := wm.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Errorf("Flush前のReadAll件数 = %d, want 0", len(records))
	}
}

func TestNewWALManagerInvalidPath(t *testing.T) {
	_, err := NewWALManager("/nonexistent/dir/test.wal")
	if err == nil {
		t.Error("存在しないディレクトリでエラーが返らなかった")
	}
}

func TestNewWALManagerEmptyFile(t *testing.T) {
	f, err := os.CreateTemp("", "empty_wal_*.wal")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil && !os.IsNotExist(err) {
			t.Errorf("cleanup: %v", err)
		}
	}()

	wm, err := NewWALManager(f.Name())
	if err != nil {
		t.Fatalf("空ファイルでエラーが返った: %v", err)
	}
	defer func() {
		if err := wm.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	lsn, err := wm.Append(&LogRecord{TxnID: 1, Op: OpInsert})
	if err != nil {
		t.Fatal(err)
	}
	if lsn != 0 {
		t.Errorf("空ファイル後の最初のLSN = %d, want 0", lsn)
	}
}

func TestCorruptedWALFile(t *testing.T) {
	f, err := os.CreateTemp("", "corrupt_wal_*.wal")
	if err != nil {
		t.Fatal(err)
	}
	// 31バイト未満の不完全なレコードを書く（クラッシュ時を模倣）
	if _, err := f.Write([]byte{0x00, 0x01, 0x02}); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil && !os.IsNotExist(err) {
			t.Errorf("cleanup: %v", err)
		}
	}()

	// 途中で切れたレコードはクラッシュ時の残骸として無視される
	wm, err := NewWALManager(f.Name())
	if err != nil {
		t.Fatalf("不完全なレコードでエラーが返った: %v", err)
	}
	defer func() {
		if err := wm.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	// 不完全なレコードは存在しないものとして扱われるのでnextLSN=0
	lsn, err := wm.Append(&LogRecord{TxnID: 1, Op: OpInsert})
	if err != nil {
		t.Fatal(err)
	}
	if lsn != 0 {
		t.Errorf("nextLSN = %d, want 0", lsn)
	}
}
