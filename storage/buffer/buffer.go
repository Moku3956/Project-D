package buffer

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
)

// frame はバッファプールの1スロット。
type frame struct {
	p        *page.Page
	pinCount int
	isDirty  bool
	txnID    uint64
	lruElem  *list.Element
}

// BufferPool はページを保持するバッファプール。LRU + No-Steal。
type BufferPool struct {
	mu      sync.Mutex
	disk    *page.DiskManager
	wm      *wal.WALManager
	frames  map[uint32]*frame
	lru     *list.List // front = most recently used
	maxSize int
}

func NewBufferPool(disk *page.DiskManager, wm *wal.WALManager, maxSize int) *BufferPool {
	return &BufferPool{
		disk:    disk,
		wm:      wm,
		frames:  make(map[uint32]*frame),
		lru:     list.New(),
		maxSize: maxSize,
	}
}

// FetchPage はpageIDのページをバッファプールまたはディスクから取得する。
// 呼び出し元はUnpinPageで解放する責任を持つ。
func (bp *BufferPool) FetchPage(pageID uint32) (*page.Page, error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if f, ok := bp.frames[pageID]; ok {
		f.pinCount++
		bp.lru.MoveToFront(f.lruElem)
		return f.p, nil
	}

	if len(bp.frames) >= bp.maxSize {
		if err := bp.evict(); err != nil {
			return nil, err
		}
	}

	p, err := bp.disk.ReadPage(pageID)
	if err != nil {
		return nil, err
	}

	f := &frame{p: p, pinCount: 1}
	f.lruElem = bp.lru.PushFront(pageID)
	bp.frames[pageID] = f
	return p, nil
}

// UnpinPage はページの使用終了を通知する。isDirty=trueで変更ありを示す。
func (bp *BufferPool) UnpinPage(pageID uint32, isDirty bool, txnID uint64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	f, ok := bp.frames[pageID]
	if !ok {
		return
	}
	if f.pinCount > 0 {
		f.pinCount--
	}
	if isDirty {
		f.isDirty = true
		f.txnID = txnID
	}
}

// FlushPage はdirtyページをディスクに書き出す（WALフラッシュ後）。
func (bp *BufferPool) FlushPage(pageID uint32) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	f, ok := bp.frames[pageID]
	if !ok || !f.isDirty {
		return nil
	}
	if err := bp.wm.Flush(); err != nil {
		return err
	}
	if err := bp.disk.WritePage(f.p); err != nil {
		return err
	}
	f.isDirty = false
	return nil
}

// FlushAll はコミット済みの全dirtyページをディスクに書き出す。
func (bp *BufferPool) FlushAll(committedTxns map[uint64]bool) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if err := bp.wm.Flush(); err != nil {
		return err
	}
	for _, f := range bp.frames {
		if !f.isDirty {
			continue
		}
		if !committedTxns[f.txnID] {
			continue
		}
		if err := bp.disk.WritePage(f.p); err != nil {
			return err
		}
		f.isDirty = false
	}
	return nil
}

// evict はNo-Steal制約を守りながらLRUページを追い出す。
func (bp *BufferPool) evict() error {
	for elem := bp.lru.Back(); elem != nil; elem = elem.Prev() {
		pageID := elem.Value.(uint32)
		f := bp.frames[pageID]
		if f.pinCount > 0 || f.isDirty {
			// pinされているか未コミットdirtyはevict不可
			continue
		}
		bp.lru.Remove(elem)
		delete(bp.frames, pageID)
		return nil
	}
	return fmt.Errorf("buffer pool: no evictable frame")
}
