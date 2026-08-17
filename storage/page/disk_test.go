package page

import (
	"os"
	"testing"
)

func tempDB(t *testing.T) (string, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "disk_test_*.db")
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

func TestNewDiskManagerCreatesHeader(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := dm.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	if dm.RootPageID() != NoPageID {
		t.Errorf("RootPageID = %d, want %d", dm.RootPageID(), NoPageID)
	}
}

func TestAllocatePageIncrementsID(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := dm.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	p0, err := dm.AllocatePage(TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}
	p1, err := dm.AllocatePage(TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}

	if p0.PageID() != 0 {
		t.Errorf("1枚目のPageID = %d, want 0", p0.PageID())
	}
	if p1.PageID() != 1 {
		t.Errorf("2枚目のPageID = %d, want 1", p1.PageID())
	}
}

func TestWriteAndReadPage(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := dm.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	p, err := dm.AllocatePage(TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}
	p.SetLSN(42)
	if err := dm.WritePage(p); err != nil {
		t.Fatal(err)
	}

	got, err := dm.ReadPage(p.PageID())
	if err != nil {
		t.Fatal(err)
	}
	if got.LSN() != 42 {
		t.Errorf("LSN = %d, want 42", got.LSN())
	}
}

func TestSetAndGetRootPageID(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := dm.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	if err := dm.SetRootPageID(7); err != nil {
		t.Fatal(err)
	}
	if dm.RootPageID() != 7 {
		t.Errorf("RootPageID = %d, want 7", dm.RootPageID())
	}
}

func TestReopenRestoresState(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dm.AllocatePage(TypeLeaf); err != nil {
		t.Fatal(err)
	}
	if _, err := dm.AllocatePage(TypeLeaf); err != nil {
		t.Fatal(err)
	}
	if err := dm.SetRootPageID(1); err != nil {
		t.Fatal(err)
	}
	if err := dm.Close(); err != nil {
		t.Fatal(err)
	}

	dm2, err := NewDiskManager(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := dm2.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	if dm2.RootPageID() != 1 {
		t.Errorf("RootPageID = %d, want 1", dm2.RootPageID())
	}
	p, err := dm2.AllocatePage(TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}
	if p.PageID() != 2 {
		t.Errorf("再オープン後のPageID = %d, want 2", p.PageID())
	}
}

func TestNewDiskManagerInvalidPath(t *testing.T) {
	_, err := NewDiskManager("/nonexistent/dir/test.db")
	if err == nil {
		t.Error("存在しないディレクトリでエラーが返らなかった")
	}
}

func TestNewDiskManagerBadMagic(t *testing.T) {
	f, err := os.CreateTemp("", "bad_magic_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil && !os.IsNotExist(err) {
			t.Errorf("cleanup: %v", err)
		}
	}()

	if _, err := f.Write(make([]byte, FileHeaderSize)); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = NewDiskManager(f.Name())
	if err == nil {
		t.Error("不正なマジックナンバーでエラーが返らなかった")
	}
}
