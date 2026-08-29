package buffer

import (
	"os"
	"testing"

	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
)

func setupPool(t *testing.T, maxSize int) (*BufferPool, func()) {
	t.Helper()

	dbPath := t.TempDir() + "/test.db"
	walPath := t.TempDir() + "/test.wal"

	dm, err := page.NewDiskManager(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	wm, err := wal.NewWALManager(walPath)
	if err != nil {
		t.Fatal(err)
	}

	bp := NewBufferPool(dm, wm, maxSize)

	return bp, func() {
		if err := dm.Close(); err != nil {
			t.Errorf("dm.Close: %v", err)
		}
		if err := wm.Close(); err != nil {
			t.Errorf("wm.Close: %v", err)
		}
		if err := os.Remove(dbPath); err != nil {
			t.Errorf("os.Remove(dbPath): %v", err)
		}
		if err := os.Remove(walPath); err != nil {
			t.Errorf("os.Remove(walPath): %v", err)
		}
	}
}

// ---- 正常系 ----

func TestFetchPageCacheMiss(t *testing.T) {
	bp, cleanup := setupPool(t, 10)
	defer cleanup()

	p, err := bp.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}
	pageID := p.PageID()

	got, err := bp.FetchPage(pageID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PageID() != pageID {
		t.Errorf("PageID = %d, want %d", got.PageID(), pageID)
	}
}

func TestFetchPageCacheHit(t *testing.T) {
	bp, cleanup := setupPool(t, 10)
	defer cleanup()

	p, err := bp.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}
	pageID := p.PageID()

	first, err := bp.FetchPage(pageID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := bp.FetchPage(pageID)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("キャッシュヒット時に異なるポインタが返った")
	}
}

func TestUnpinPageDecreasesPinCount(t *testing.T) {
	bp, cleanup := setupPool(t, 10)
	defer cleanup()

	p, err := bp.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}
	pageID := p.PageID()

	if _, err := bp.FetchPage(pageID); err != nil {
		t.Fatal(err)
	}
	bp.UnpinPage(pageID, false, 0)

	f := bp.frames[pageID]
	if f.pinCount != 0 {
		t.Errorf("pinCount = %d, want 0", f.pinCount)
	}
}

func TestFlushPageWritesToDisk(t *testing.T) {
	bp, cleanup := setupPool(t, 10)
	defer cleanup()

	p, err := bp.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}
	pageID := p.PageID()

	fetched, err := bp.FetchPage(pageID)
	if err != nil {
		t.Fatal(err)
	}
	fetched.SetLSN(99)
	bp.UnpinPage(pageID, true, 1)

	if err := bp.FlushPage(pageID); err != nil {
		t.Fatal(err)
	}

	got, err := bp.disk.ReadPage(pageID)
	if err != nil {
		t.Fatal(err)
	}
	if got.LSN() != 99 {
		t.Errorf("LSN = %d, want 99", got.LSN())
	}

	f := bp.frames[pageID]
	if f.isDirty {
		t.Error("FlushPage後もisDirtyがtrueのまま")
	}
}

func TestFlushAllOnlyCommitted(t *testing.T) {
	bp, cleanup := setupPool(t, 10)
	defer cleanup()

	p0, err := bp.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}
	p1, err := bp.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}

	f0, err := bp.FetchPage(p0.PageID())
	if err != nil {
		t.Fatal(err)
	}
	f0.SetLSN(10)
	bp.UnpinPage(p0.PageID(), true, 1) // txnID=1 コミット済み

	f1, err := bp.FetchPage(p1.PageID())
	if err != nil {
		t.Fatal(err)
	}
	f1.SetLSN(20)
	bp.UnpinPage(p1.PageID(), true, 2) // txnID=2 未コミット

	committed := map[uint64]bool{1: true}
	if err := bp.FlushAll(committed); err != nil {
		t.Fatal(err)
	}

	got0, err := bp.disk.ReadPage(p0.PageID())
	if err != nil {
		t.Fatal(err)
	}
	if got0.LSN() != 10 {
		t.Errorf("txnID=1のページLSN = %d, want 10", got0.LSN())
	}

	got1, err := bp.disk.ReadPage(p1.PageID())
	if err != nil {
		t.Fatal(err)
	}
	if got1.LSN() == 20 {
		t.Error("未コミットのtxnID=2のページがディスクに書かれてしまった")
	}
}

func TestEvictLRU(t *testing.T) {
	bp, cleanup := setupPool(t, 2)
	defer cleanup()

	p0, err := bp.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}
	p1, err := bp.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := bp.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := bp.FetchPage(p0.PageID()); err != nil {
		t.Fatal(err)
	}
	bp.UnpinPage(p0.PageID(), false, 0)

	if _, err := bp.FetchPage(p1.PageID()); err != nil {
		t.Fatal(err)
	}
	bp.UnpinPage(p1.PageID(), false, 0)

	// maxSize=2に達した状態でp2をfetchするとp0が追い出される
	if _, err := bp.FetchPage(p2.PageID()); err != nil {
		t.Fatal(err)
	}
	bp.UnpinPage(p2.PageID(), false, 0)

	if _, ok := bp.frames[p0.PageID()]; ok {
		t.Error("LRU末尾のp0が追い出されなかった")
	}
}

// ---- 異常系 ----

func TestEvictFailsWhenAllPinned(t *testing.T) {
	bp, cleanup := setupPool(t, 1)
	defer cleanup()

	p0, err := bp.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}
	p1, err := bp.disk.AllocatePage(page.TypeLeaf)
	if err != nil {
		t.Fatal(err)
	}

	// p0をpinしたままにする
	if _, err := bp.FetchPage(p0.PageID()); err != nil {
		t.Fatal(err)
	}

	// p1をfetchしようとするとevictできずエラー
	_, err = bp.FetchPage(p1.PageID())
	if err == nil {
		t.Error("全ページpinされているときにエラーが返らなかった")
	}
}

func TestFetchPageInvalidPageID(t *testing.T) {
	bp, cleanup := setupPool(t, 10)
	defer cleanup()

	_, err := bp.FetchPage(999)
	if err == nil {
		t.Error("存在しないpageIDでエラーが返らなかった")
	}
}
