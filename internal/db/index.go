package db

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sync"

	mmapgo "github.com/edsrzf/mmap-go"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"go.uber.org/zap"
)

var (
	// ErrIndexCorrupt ErrIndexCorrupt
	ErrIndexCorrupt = errors.New("corrupt index file")
	// Encoding Encoding
	Encoding = binary.BigEndian
)

// Entry Entry
type Entry struct {
	RelativeOffset uint32
	Position       uint32
}

// Index Index
type Index struct {
	mu         sync.RWMutex
	position   int64
	entrySize  int
	baseOffset int64
	file       *os.File
	mmap       mmapgo.MMap
	limlog.Log
	maxBytes    int64
	maxEntryNum int64
	warmEntries int64 // 热点日志条
	totalSize   int64
}

// NewIndex NewIndex
func NewIndex(path string, baseOffset int64) *Index {

	idx := &Index{
		entrySize:  binary.Size(Entry{}),
		maxBytes:   10 * 1024 * 1024,
		Log:        limlog.NewLIMLog(fmt.Sprintf("Index[%s]", path)),
		baseOffset: baseOffset,
	}

	idx.maxEntryNum = idx.maxBytes / int64(idx.entrySize)
	idx.warmEntries = 8192 / int64(idx.entrySize)
	var err error
	idx.file, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		idx.Error("open file failed", zap.Error(err))
		panic(err)
	}
	fi, err := idx.file.Stat()
	if err != nil {
		idx.Error("stat file failed", zap.Error(err))
		panic(err)
	} else if fi.Size() > 0 {
		idx.totalSize = fi.Size()
		idx.position = fi.Size()
	}

	if err := idx.file.Truncate(roundDown(idx.maxBytes, int64(idx.entrySize))); err != nil {
		idx.Error("Truncate file failed", zap.Error(err))
		panic(err)
	}
	idx.mmap, err = mmapgo.Map(idx.file, mmapgo.RDWR, 0)
	if err != nil {
		panic(errors.New("mmap file failed"))
	}
	err = idx.resetPosistion()
	if err != nil {
		panic(err)
	}

	return idx
}

func (idx *Index) resetPosistion() error {
	var position int64 = 0
	for {
		if position >= idx.totalSize {
			break
		}
		entry := new(Entry)
		if err := idx.readEntryAtPosition(entry, position); err != nil {
			return err
		}
		if entry.RelativeOffset == 0 {
			break
		}
		position += int64(idx.entrySize)
	}
	idx.position = position
	return nil
}

// Append Append
func (idx *Index) Append(offset int64, position int64) error {
	b := new(bytes.Buffer)
	if err := binary.Write(b, Encoding, Entry{
		RelativeOffset: uint32(offset - int64(idx.baseOffset)),
		Position:       uint32(position),
	}); err != nil {
		return err
	}
	if idx.position >= idx.maxBytes {
		idx.Warn("Index file is full, give up！")
		return nil
	}
	idx.writeAt(b.Bytes(), idx.position)
	idx.mu.Lock()
	idx.position += int64(idx.entrySize)
	idx.mu.Unlock()
	return nil
}

// Lookup  Find the largest offset less than or equal to the given targetOffset
// and return a pair holding this offset and its corresponding physical file position
func (idx *Index) Lookup(targetOffset int64) (OffsetPosition, error) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	min, _ := idx.indexSlotRangeFor(targetOffset)
	if min == -1 {
		return OffsetPosition{
			Offset:   idx.baseOffset,
			Position: 0,
		}, nil
	}
	return idx.parseEntry(min), nil
}

// TruncateEntries TruncateEntries
func (idx *Index) TruncateEntries(number int) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if int64(number*idx.entrySize) > idx.position {
		return errors.New("bad truncate number")
	}
	idx.position = int64(number * idx.entrySize)
	return nil
}

// IsFull Is full
func (idx *Index) IsFull() bool {
	return idx.position >= idx.maxBytes
}

// WriteAt WriteAt
func (idx *Index) writeAt(p []byte, offset int64) (n int) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	return copy(idx.mmap[offset:offset+int64(idx.entrySize)], p)
}

// Sync Sync
func (idx *Index) Sync() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if err := idx.file.Sync(); err != nil {
		return errors.New("file sync failed")
	}
	if err := idx.mmap.Flush(); err != nil {
		return errors.New("mmap flush failed")
	}
	return nil
}

// Close Close
func (idx *Index) Close() error {
	if err := idx.Sync(); err != nil {
		return err
	}
	if err := idx.file.Truncate(idx.position); err != nil {
		return err
	}
	if err := idx.file.Close(); err != nil {
		return err
	}
	if err := idx.mmap.Unmap(); err != nil {
		return err
	}
	return nil
}

// SanityCheck Sanity check
func (idx *Index) SanityCheck() error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if idx.position == 0 {
		return nil
	} else if idx.position%int64(idx.entrySize) != 0 {
		return ErrIndexCorrupt
	} else {
		//read last entry
		entry := new(Entry)
		if err := idx.readEntryAtPosition(entry, idx.position-int64(idx.entrySize)); err != nil {
			return err
		}
		if idx.getRealOffset(entry) < idx.baseOffset {
			return ErrIndexCorrupt
		}
		return nil
	}
}

// ReadEntryAtPosition ReadEntryAtPosition
func (idx *Index) readEntryAtPosition(entry *Entry, position int64) error {
	p := make([]byte, idx.entrySize)
	if _, err := idx.readAtPosition(p, position); err != nil {
		return err
	}
	b := bytes.NewReader(p)
	err := binary.Read(b, Encoding, entry)
	if err != nil {
		return errors.New("binary read failed")
	}
	return nil
}

// ReadAt ReadAt
func (idx *Index) readAtPosition(p []byte, position int64) (n int, err error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if idx.position < position+int64(idx.entrySize) {
		return 0, io.EOF
	}
	n = copy(p, idx.mmap[position:position+int64(idx.entrySize)])
	return n, nil
}

func (idx *Index) getRealOffset(entry *Entry) int64 {
	return int64(entry.RelativeOffset) + idx.baseOffset
}

func (idx *Index) indexSlotRangeFor(target int64) (int64, int64) {
	entries := idx.position / int64(idx.entrySize)
	if entries == 0 {
		return -1, -1
	}
	var binarySearch = func(begin, end int64) (int64, int64) {
		var lo = begin
		var hi = end
		for lo < hi {
			var mid = (lo + hi + 1) >> 1
			var found = idx.parseEntry(mid)
			var compareResult = idx.compareIndexEntry(found, target)
			if compareResult > 0 {
				hi = mid - 1
			} else if compareResult < 0 {
				lo = mid
			} else {
				return mid, mid
			}

		}
		if lo == entries-1 {
			return lo, -1
		}
		return lo, lo + 1
	}
	var firstHotEntry = int64(math.Max(0, float64(entries-1-idx.warmEntries)))
	if idx.compareIndexEntry(idx.parseEntry(firstHotEntry), target) < 0 {
		return binarySearch(firstHotEntry, entries-1)
	}
	if idx.compareIndexEntry(idx.parseEntry(0), target) > 0 {
		return -1, 0
	}
	return binarySearch(0, firstHotEntry)

}

func (idx *Index) compareIndexEntry(indexEntry OffsetPosition, target int64) int {

	if indexEntry.Offset > target {
		return 1
	} else if indexEntry.Offset < target {
		return -1
	}
	return 0
}

func (idx *Index) parseEntry(mid int64) OffsetPosition {
	p := make([]byte, idx.entrySize)
	position := mid * int64(idx.entrySize)
	copyEnd := position + int64(idx.entrySize)
	if copyEnd > int64(len(idx.mmap)) {
		return OffsetPosition{
			Offset: idx.baseOffset,
		}
	}
	copy(p, idx.mmap[position:copyEnd])
	b := bytes.NewReader(p)
	var entry = &Entry{}
	err := binary.Read(b, Encoding, entry)
	if err != nil {
		idx.Error("binary read failed", zap.Error(err))
		panic(err)
	}
	return OffsetPosition{
		Offset:   idx.baseOffset + int64(entry.RelativeOffset),
		Position: int64(entry.Position),
	}
}

// OffsetPosition OffsetPosition
type OffsetPosition struct {
	Offset   int64
	Position int64
}
