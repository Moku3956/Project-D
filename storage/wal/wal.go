package wal

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

type Operation uint8

const (
	OpInsert Operation = 0x01
	OpUpdate Operation = 0x02
	OpDelete Operation = 0x03
	OpCommit Operation = 0x04
	OpAbort  Operation = 0x05
)

// LogRecord はWALの1レコード。
type LogRecord struct {
	LSN      uint64
	TxnID    uint64
	PageID   uint32
	Op       Operation
	PrevLSN  uint64
	RedoData []byte
}

// WALManager はログの追記・フラッシュ・リカバリを担う。
type WALManager struct {
	mu      sync.Mutex
	file    *os.File
	nextLSN uint64
	buf     []byte
}

func NewWALManager(path string) (*WALManager, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	wm := &WALManager{file: file}
	if info.Size() > 0 {
		if err := wm.scanNextLSN(); err != nil {
			return nil, err
		}
	}
	return wm, nil
}

func (wm *WALManager) Close() error { return wm.file.Close() }

// Append はログレコードをバッファに追記しLSNを返す。
func (wm *WALManager) Append(rec *LogRecord) (uint64, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	lsn := wm.nextLSN
	wm.nextLSN++
	rec.LSN = lsn

	wm.buf = append(wm.buf, encode(rec)...)
	return lsn, nil
}

// Flush はバッファをディスクに書き出す。
func (wm *WALManager) Flush() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if len(wm.buf) == 0 {
		return nil
	}
	if _, err := wm.file.Write(wm.buf); err != nil {
		return err
	}
	if err := wm.file.Sync(); err != nil {
		return err
	}
	wm.buf = wm.buf[:0]
	return nil
}

// ReadAll はログファイルの全レコードを返す（リカバリ用）。
func (wm *WALManager) ReadAll() ([]*LogRecord, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, err := wm.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	var records []*LogRecord
	for {
		rec, err := decodeOne(wm.file)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("wal read: %w", err)
		}
		records = append(records, rec)
	}
	return records, nil
}

// scanNextLSN は既存ログを読んでnextLSNを初期化する。
func (wm *WALManager) scanNextLSN() error {
	records, err := wm.ReadAll()
	if err != nil {
		return err
	}
	if len(records) > 0 {
		wm.nextLSN = records[len(records)-1].LSN + 1
	}
	return nil
}

// ---- エンコード / デコード ----
//
// レコードフォーマット:
//   LSN(8) TxnID(8) PageID(4) Op(1) PrevLSN(8) RedoSize(2) RedoData(n)

func encode(rec *LogRecord) []byte {
	size := 8 + 8 + 4 + 1 + 8 + 2 + len(rec.RedoData)
	buf := make([]byte, size)
	binary.BigEndian.PutUint64(buf[0:8], rec.LSN)
	binary.BigEndian.PutUint64(buf[8:16], rec.TxnID)
	binary.BigEndian.PutUint32(buf[16:20], rec.PageID)
	buf[20] = byte(rec.Op)
	binary.BigEndian.PutUint64(buf[21:29], rec.PrevLSN)
	binary.BigEndian.PutUint16(buf[29:31], uint16(len(rec.RedoData)))
	copy(buf[31:], rec.RedoData)
	return buf
}

func decodeOne(r io.Reader) (*LogRecord, error) {
	header := make([]byte, 31)
	if _, err := io.ReadFull(r, header); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, io.EOF
		}
		return nil, err
	}
	rec := &LogRecord{
		LSN:     binary.BigEndian.Uint64(header[0:8]),
		TxnID:   binary.BigEndian.Uint64(header[8:16]),
		PageID:  binary.BigEndian.Uint32(header[16:20]),
		Op:      Operation(header[20]),
		PrevLSN: binary.BigEndian.Uint64(header[21:29]),
	}
	redoSize := binary.BigEndian.Uint16(header[29:31])
	if redoSize > 0 {
		rec.RedoData = make([]byte, redoSize)
		if _, err := io.ReadFull(r, rec.RedoData); err != nil {
			return nil, err
		}
	}
	return rec, nil
}
