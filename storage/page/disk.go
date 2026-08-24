package page

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	magicNumber    = "MOKU"
	fileVersion    = uint16(1)
	headerPageSize = uint16(PageSize)

	// NoPageID はページが存在しないことを示す番兵値。
	NoPageID = uint32(0xFFFFFFFF)
)

// DiskManager はファイルへのページ読み書きを担う。
type DiskManager struct {
	file       *os.File
	rootPageID uint32
	nextPageID uint32
}

// NewDiskManager はファイルを開き、新規ならヘッダを書き込み、既存ならヘッダを読み込んでDiskManagerを返す。
func NewDiskManager(path string) (*DiskManager, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	dm := &DiskManager{
		file:       file,
		rootPageID: NoPageID,
	}

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if info.Size() == 0 {
		if err := dm.writeFileHeader(NoPageID); err != nil {
			return nil, err
		}
	} else {
		if err := dm.readFileHeader(); err != nil {
			return nil, err
		}
	}
	return dm, nil
}

// Close はファイルを閉じる。
func (dm *DiskManager) Close() error { return dm.file.Close() }

// AllocatePage は新しいページIDを確保してディスクに空ページを書く。
func (dm *DiskManager) AllocatePage(pageType uint8) (*Page, error) {
	id := dm.nextPageID
	dm.nextPageID++
	p := NewPage(pageType, id)
	if err := dm.WritePage(p); err != nil {
		return nil, err
	}
	if err := dm.writeFileHeader(dm.RootPageID()); err != nil {
		return nil, err
	}
	return p, nil
}

// ReadPage はpageIDのページをディスクから読む。
func (dm *DiskManager) ReadPage(pageID uint32) (*Page, error) {
	buf := make([]byte, PageSize)
	offset := int64(FileHeaderSize) + int64(pageID)*int64(PageSize)
	if _, err := dm.file.ReadAt(buf, offset); err != nil {
		return nil, fmt.Errorf("read page %d: %w", pageID, err)
	}
	return FromBytes(buf), nil
}

// WritePage はページをディスクに書く。
func (dm *DiskManager) WritePage(p *Page) error {
	offset := int64(FileHeaderSize) + int64(p.PageID())*int64(PageSize)
	if _, err := dm.file.WriteAt(p.Bytes(), offset); err != nil {
		return fmt.Errorf("write page %d: %w", p.PageID(), err)
	}
	return nil
}

// ファイルヘッダのレイアウト（先頭4KB）:
// 0-3:   マジックナンバー "MYDB"
// 4-5:   バージョン (uint16)
// 6-7:   ページサイズ (uint16)
// 8-11:  ルートページID (uint32)
// 12-15: 次のページID (uint32)

// writeFileHeader はルートページIDとnextPageIDをファイル先頭のヘッダ領域に書き込む。
func (dm *DiskManager) writeFileHeader(rootPageID uint32) error {
	buf := make([]byte, FileHeaderSize)
	copy(buf[0:4], magicNumber)
	binary.BigEndian.PutUint16(buf[4:6], fileVersion)
	binary.BigEndian.PutUint16(buf[6:8], headerPageSize)
	binary.BigEndian.PutUint32(buf[8:12], rootPageID)
	binary.BigEndian.PutUint32(buf[12:16], dm.nextPageID)
	_, err := dm.file.WriteAt(buf, 0)
	if err == nil {
		dm.rootPageID = rootPageID
	}
	return err
}

// readFileHeader はファイル先頭のヘッダ領域からルートページIDとnextPageIDを読み込む。
func (dm *DiskManager) readFileHeader() error {
	buf := make([]byte, FileHeaderSize)
	if _, err := dm.file.ReadAt(buf, 0); err != nil {
		return err
	}
	if string(buf[0:4]) != magicNumber {
		return fmt.Errorf("invalid file: bad magic number")
	}
	dm.rootPageID = binary.BigEndian.Uint32(buf[8:12])
	dm.nextPageID = binary.BigEndian.Uint32(buf[12:16])
	return nil
}

// RootPageID はB+Treeの根ノードのページIDを返す。
func (dm *DiskManager) RootPageID() uint32 {
	return dm.rootPageID
}

// SetRootPageID はB+Treeの根ノードのページIDを更新してヘッダに書き込む。
func (dm *DiskManager) SetRootPageID(id uint32) error {
	return dm.writeFileHeader(id)
}

// Sync はOSのバッファをディスクにフラッシュする。
func (dm *DiskManager) Sync() error { return dm.file.Sync() }
