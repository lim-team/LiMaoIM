package db

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"go.uber.org/zap"
)

var (
	// ErrorNotData ErrorNotData
	ErrorNotData = errors.New("no data")

	// MagicNumber MagicNumber
	MagicNumber = [2]byte{0x15, 0x16} // lm
	// EndMagicNumber EndMagicNumber
	EndMagicNumber = [1]byte{0x3}
	// LogVersion log version
	LogVersion = [1]byte{0x01}
	// SnapshotMagicNumber SnapshotMagicNumber
	SnapshotMagicNumber = [2]byte{0xb, 0xa} // ba
	// EndSnapshotMagicNumber EndSnapshotMagicNumber
	EndSnapshotMagicNumber = [1]byte{0xf}
)

const (
	// OffsetSize OffsetSize
	OffsetSize = 8
	// LogDataLenSize LogDataLenSize
	LogDataLenSize = 4
	// AppliIndexSize AppliIndexSize
	AppliIndexSize = 8
	// LogMaxSize  Maximum size of a single log data
	LogMaxSize = 1024 * 1024
)

// Segment Segment
type Segment struct {
	topicDir   string
	logDir     string
	baseOffset int64
	index      *Index
	limlog.Log
	logWriter io.Writer
	logFile   *os.File
	// logMMap mmapgo.MMap
	sync.Mutex
	position                 int64
	indexIntervalBytes       int64
	bytesSinceLastIndexEntry int64
}

// NewSegment NewSegment
func NewSegment(topicDir string, baseOffset int64) *Segment {
	logDir := filepath.Join(topicDir, "logs")
	s := &Segment{
		logDir:             logDir,
		topicDir:           topicDir,
		baseOffset:         baseOffset,
		indexIntervalBytes: 4 * 1024,
	}
	err := os.MkdirAll(s.logDir, FileDefaultMode)
	if err != nil {
		s.Error("mkdir segment dir fail", zap.Error(err))
		panic(err)
	}
	s.Log = limlog.NewLIMLog(fmt.Sprintf("Segment[%s]", s.logPath()))
	s.index = NewIndex(s.indexPath(), baseOffset)
	pathStr := s.logPath()
	s.logFile, err = os.OpenFile(pathStr, os.O_RDWR|os.O_CREATE|os.O_APPEND, FileDefaultMode)
	if err != nil {
		s.Error("open file fail!", zap.Error(err), zap.String("path", pathStr))
		panic(err)
	}

	s.logWriter = s.logFile
	return s
}

// RebuildIndex RebuildIndex
func (s *Segment) rebuildIndex() error {
	// 	if err := s.index.SanityCheck(); err != nil {
	// 		return err
	// 	}
	// 	if err := s.index.TruncateEntries(0); err != nil {
	// 		return err
	// 	}
	// 	_, err := s.logFile.Seek(0, 0)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	b := new(bytes.Buffer)

	// 	nextOffset := s.baseOffset
	// 	position := int64(0)
	// loop:
	// 	for {
	// 		// get offset and size
	// 		_, err = io.CopyN(b, s.logFile, 8)
	// 		if err != nil {
	// 			break loop
	// 		}
	// 		_, err = io.CopyN(b, s.logFile, 4)
	// 		if err != nil {
	// 			break loop
	// 		}
	// 	}
	return nil
}

// SanityCheck Sanity check
func (s *Segment) SanityCheck() (int64, error) {

	stat, err := s.logFile.Stat()
	if err != nil {
		s.Error("Stat file fail!", zap.Error(err))
		panic(err)
	}
	segmentSizeOfByte := stat.Size()

	if segmentSizeOfByte == 0 {
		return 0, nil
	}
	hasEndMgNumer, err := s.hasEndMagicNumer(segmentSizeOfByte)
	if err != nil {
		return 0, err
	}
	if !hasEndMgNumer {
		s.Debug("No magic number at the end,sanity check mode is full")
		return s.sanityFullCheck(segmentSizeOfByte)
	}
	if segmentSizeOfByte <= LogMaxSize+s.logMinLen() {
		s.Debug("File is too small,sanity check mode is full")
		return s.sanityFullCheck(segmentSizeOfByte)
	}

	s.Debug("sanity check mode is simple")
	check, lastMsgStartPosition, err := s.sanitySimpleCheck(segmentSizeOfByte)
	if err != nil {
		return 0, err
	}
	if !check {
		s.Debug("sanity check simple mode is fail！Turn on full mode")
		return s.sanityFullCheck(segmentSizeOfByte)
	}
	return lastMsgStartPosition, nil
}

func (s *Segment) sanityFullCheck(segmentSizeOfByte int64) (int64, error) {

	return s.check(0, segmentSizeOfByte, true)
}

func (s *Segment) sanitySimpleCheck(segmentSizeOfByte int64) (bool, int64, error) {

	warmPosition := LogMaxSize + s.logMinLen()
	for {
		if warmPosition >= segmentSizeOfByte {
			return false, 0, nil
		}
		has, num, err := s.hasMagicNumer(warmPosition)
		if err != nil {
			return false, 0, err
		}
		if has {
			break
		}
		warmPosition += int64(num)

	}
	lastMsgStartPosition, err := s.check(warmPosition, segmentSizeOfByte, false)
	if err != nil {
		return false, 0, err
	}
	return true, lastMsgStartPosition, nil
}

func (s *Segment) hasEndMagicNumer(segmentSizeOfByte int64) (bool, error) {
	var p = make([]byte, 1)
	_, err := s.logFile.ReadAt(p, segmentSizeOfByte-1)
	if err != nil {
		return false, err
	}
	return bytes.Equal(p, EndMagicNumber[:]), nil
}

func (s *Segment) hasMagicNumer(position int64) (bool, int, error) {
	var v = make([]byte, 2)
	n, err := s.logFile.ReadAt(v, position)
	if err != nil {
		return false, 0, err
	}
	return bytes.Equal(v, MagicNumber[:]), n, nil
}

func (s *Segment) check(startPosition, segmentSizeOfByte int64, keepCorrect bool) (int64, error) {
	_, err := s.logFile.Seek(startPosition, os.SEEK_SET)
	if err != nil {
		return 0, err
	}
	b := new(bytes.Buffer)

	var byteOffset int64 = startPosition
	var lastMsgStartPosition int64 = startPosition
	for {
		if byteOffset >= segmentSizeOfByte {
			break
		}
		lastMsgStartPosition = byteOffset
		_, err = io.CopyN(b, s.logFile, int64(len(MagicNumber)))
		if err != nil {
			break
		}
		byteOffset += int64(len(MagicNumber))
		if !bytes.Equal(b.Bytes(), MagicNumber[:]) {
			s.Warn("The magic number does not match the log is wrong！", zap.ByteString("expect", MagicNumber[:]), zap.ByteString("actual", b.Bytes()))
			break
		}
		if byteOffset+int64(len(LogVersion)+LogDataLenSize) >= segmentSizeOfByte {
			break
		}
		_, err = io.CopyN(b, s.logFile, int64(len(LogVersion)+LogDataLenSize))
		if err != nil {
			break
		}

		byteOffset += int64(len(LogVersion) + LogDataLenSize)

		dataLenBytes := b.Bytes()[(len(MagicNumber) + len(LogVersion)):(len(MagicNumber) + len(LogVersion) + LogDataLenSize)]
		size := int64(Encoding.Uint32(dataLenBytes))

		if byteOffset+size+int64(OffsetSize+AppliIndexSize+len(EndMagicNumber)) > segmentSizeOfByte {
			break
		}
		byteOffset += (size + int64(OffsetSize+AppliIndexSize+len(EndMagicNumber)))

		s.position = byteOffset

		b.Truncate(0)

		_, err := s.logFile.Seek(byteOffset, os.SEEK_SET)
		if err != nil {
			break
		}
	}

	if keepCorrect {
		if s.position != segmentSizeOfByte {
			s.Warn("Back up the original log and remove the damaged log")
			err = s.backup()
			if err != nil {
				s.Error("backup fail!", zap.Error(err))
				return 0, err
			}

			err = s.logFile.Truncate(s.position)
			if err != nil {
				s.Error("truncate fail", zap.Error(err))
				return 0, err
			}
			_, err := s.logFile.Seek(s.position, os.SEEK_SET)
			if err != nil {
				return 0, err
			}
		}
	}

	return lastMsgStartPosition, nil
}

func (s *Segment) backup() error {
	_, err := CopyFile(s.backupPath(time.Now().UnixNano()), s.logPath())
	return err
}

// AppendLog AppendLog
func (s *Segment) AppendLog(log ILog) (int, error) {
	data, err := log.Encode()
	if err != nil {
		return 0, err
	}
	if len(data) > LogMaxSize {
		return 0, errors.New(fmt.Sprintf("Log data is too large！can not exceed %d byte", LogMaxSize))
	}
	s.Lock()
	defer s.Unlock()

	n, err := s.append(s.wrapLogData(log.Offset(), log.GetAppliIndex(), data))
	if err != nil {
		return 0, err
	}

	if s.bytesSinceLastIndexEntry > s.indexIntervalBytes {
		err = s.index.Append(log.Offset(), int64(s.position))
		if err != nil {
			return 0, err
		}
		s.bytesSinceLastIndexEntry = 0
	}

	s.bytesSinceLastIndexEntry += int64(n)

	return n, nil
}

// Sync Sync
func (s *Segment) Sync() error {
	return s.logFile.Sync()
}

// ReadLogs ReadLogs
func (s *Segment) ReadLogs(offset int64, limit uint64, callback func(data []byte) error) error {
	offsetPosition, err := s.index.Lookup(offset)
	if err != nil {
		return err
	}
	var startPosition int64 = 0
	if offsetPosition.Offset == offset {
		startPosition = offsetPosition.Position
	} else {
		startPosition, _, err = s.readTargetPosition(offsetPosition.Position, offset)
		if err != nil {
			if errors.Is(err, ErrorNotData) {
				return nil
			}
			return err
		}
	}
	return s.ReadLogsAtPosition(startPosition, limit, callback)

}

// ReadLogsAtPosition ReadLogsAtPosition
func (s *Segment) ReadLogsAtPosition(position int64, limit uint64, callback func(data []byte) error) error {
	var count uint64 = 0
	var data []byte
	var startPosition = position
	var err error
	for {
		if startPosition >= s.position || count >= limit {
			break
		}
		data, startPosition, err = s.readLogDataAtPosition(startPosition)
		if err != nil {
			return err
		}
		if callback != nil {
			err = callback(data)
			if err != nil {
				return err
			}
		}
		count++
	}
	return nil
}

// ReadAt ReadAt
func (s *Segment) ReadAt(offset int64, log ILog) error {
	offsetPosition, err := s.index.Lookup(offset)
	if err != nil {
		return err
	}
	targetPosition, _, err := s.readTargetPosition(offsetPosition.Position, offset)
	if err != nil {
		return err
	}
	data, _, err := s.readLogDataAtPosition(targetPosition)
	if err != nil {
		return err
	}
	return log.Decode(data)
}

func (s *Segment) readTargetPosition(startPosition int64, targetOffset int64) (int64, int64, error) {
	if startPosition >= s.position {
		return 0, 0, ErrorNotData
	}
	resultOffset, dataLen, err := s.readAtPosition(startPosition)
	if err != nil {
		return 0, 0, err
	}
	nextPosition := startPosition + s.logFixHeaderLen() + int64(dataLen+len(EndMagicNumber))
	if resultOffset == targetOffset {
		return startPosition, nextPosition, nil
	}

	targetPosition, nextP, err := s.readTargetPosition(nextPosition, targetOffset)
	if err != nil {
		return 0, 0, err
	}
	return targetPosition, nextP, nil
}

func (s *Segment) readAtPosition(position int64) (offset int64, dataLen int, err error) {
	sizeByte := make([]byte, LogDataLenSize)
	if _, err = s.logFile.ReadAt(sizeByte, position+int64(len(MagicNumber)+len(LogVersion))); err != nil {
		return
	}
	offsetByte := make([]byte, OffsetSize)
	if _, err = s.logFile.ReadAt(offsetByte, position+int64(len(MagicNumber)+len(LogVersion)+LogDataLenSize)); err != nil {
		return
	}
	dataLen = int(Encoding.Uint32(sizeByte))
	offset = int64(Encoding.Uint64(offsetByte))
	return
}

func (s *Segment) readLogDataAtPosition(position int64) (data []byte, nextPosition int64, err error) {
	_, dataLen, err := s.readAtPosition(position)
	if err != nil {
		return
	}
	dataStartPosition := position + s.logFixHeaderLen()
	data = make([]byte, dataLen)
	if _, err = s.logFile.ReadAt(data, dataStartPosition); err != nil {
		return nil, 0, err
	}

	nextPosition = position + s.logFixHeaderLen() + int64(dataLen+len(EndMagicNumber))
	return data, nextPosition, nil
}

func (s *Segment) logFixHeaderLen() int64 {
	return int64(len(MagicNumber) + len(LogVersion) + LogDataLenSize + OffsetSize + AppliIndexSize)
}
func (s *Segment) logMinLen() int64 {
	return s.logFixHeaderLen() + int64(len(EndMagicNumber))
}

func (s *Segment) wrapLogData(offset int64, appliIndex uint64, data []byte) []byte {
	p := new(bytes.Buffer)
	binary.Write(p, Encoding, MagicNumber)
	binary.Write(p, Encoding, LogVersion)
	binary.Write(p, Encoding, uint32(len(data)))
	binary.Write(p, Encoding, offset) // offsetSize = 8
	binary.Write(p, Encoding, appliIndex)
	binary.Write(p, Encoding, data)
	binary.Write(p, Encoding, EndMagicNumber)
	return p.Bytes()
}

func (s *Segment) append(data []byte) (int, error) {

	n, err := s.logWriter.Write(data)
	if err != nil {
		return 0, errors.Wrap(err, "log write failed")
	}
	s.position += int64(n)
	return n, nil
}

// Close Close
func (s *Segment) Close() error {
	err := s.Sync()
	if err != nil {
		return err
	}
	s.logFile.Close()
	s.index.Close()
	return nil
}

func (s *Segment) logPath() string {
	return filepath.Join(s.logDir, fmt.Sprintf(fileFormat, s.baseOffset, logSuffix))
}

func (s *Segment) indexPath() string {
	return filepath.Join(s.logDir, fmt.Sprintf(fileFormat, s.baseOffset, indexSuffix))
}

func (s *Segment) backupPath(t int64) string {
	return filepath.Join(s.logDir, fmt.Sprintf(fileFormat, s.baseOffset, fmt.Sprintf("%s.bak%d", logSuffix, t)))
}
