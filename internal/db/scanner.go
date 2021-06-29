package db

import (
	"bytes"
	"errors"
	"io"
	"os"

	"github.com/tangtaoit/limnet/pkg/limlog"
	"go.uber.org/zap"
)

// Scanner Scanner
type Scanner struct {
	limlog.Log
	f           *os.File
	segmentPath string
	position    int64
}

// NewScanner NewScanner
func NewScanner() *Scanner {

	return &Scanner{
		Log: limlog.NewLIMLog("Scanner"),
	}
}

// Scan Scan
func (s *Scanner) Scan(segmentPath string, endAppliIndex uint64, callback func(data []byte) error) (bool, error) {
	if segmentPath != s.segmentPath {
		var err error
		s.f, err = os.Open(segmentPath)
		if err != nil {
			return false, err
		}
		s.segmentPath = segmentPath
		s.position = 0
	}
	data, appliIndex, err := s.readFullWrapLog()
	if err != nil {
		if err == ErrorSegmentReadEnd {
			s.endScanner()
			return false, nil
		}
		s.endScanner()
		s.Error("readFullWrapLog is error", zap.Error(err), zap.String("segmentPath", segmentPath))
		return false, err
	}
	if appliIndex > endAppliIndex {
		s.endScanner()
		return false, nil
	}
	err = callback(data)
	if err != nil {
		s.endScanner()
		return false, err
	}
	return true, nil
}

func (s *Scanner) endScanner() {
	s.f.Close()
	s.f = nil
	s.segmentPath = ""
}

func (s *Scanner) readFullWrapLog() ([]byte, uint64, error) {
	var startBytes = make([]byte, len(MagicNumber))
	if _, err := s.f.ReadAt(startBytes, s.position); err != nil {
		if err == io.EOF {
			return nil, 0, ErrorSegmentReadEnd
		}
		return nil, 0, err
	}
	s.position += int64(len(startBytes))
	if !bytes.Equal(startBytes, MagicNumber[:]) {
		s.Error("Wrong magic number！", zap.ByteString("expect", MagicNumber[:]), zap.ByteString("actual", startBytes))
		return nil, 0, errors.New("Wrong magic number！")
	}
	versionBytes := make([]byte, len(LogVersion))
	if _, err := s.f.ReadAt(versionBytes, s.position); err != nil {
		return nil, 0, err
	}
	s.position += int64(len(versionBytes))
	dataLenBytes := make([]byte, LogDataLenSize)
	if _, err := s.f.ReadAt(dataLenBytes, s.position); err != nil {
		return nil, 0, err
	}
	s.position += int64(len(dataLenBytes))
	dataLen := Encoding.Uint32(dataLenBytes)

	offsetBytes := make([]byte, OffsetSize)
	if _, err := s.f.ReadAt(offsetBytes, s.position); err != nil {
		return nil, 0, err
	}
	s.position += int64(len(offsetBytes))

	appliIndexBytes := make([]byte, AppliIndexSize)
	if _, err := s.f.ReadAt(appliIndexBytes, s.position); err != nil {
		return nil, 0, err
	}
	s.position += int64(len(appliIndexBytes))

	data := make([]byte, dataLen)
	if _, err := s.f.ReadAt(data, s.position); err != nil {
		return nil, 0, err
	}
	s.position += int64(len(data))

	endBytes := make([]byte, len(EndMagicNumber))
	if _, err := s.f.ReadAt(endBytes, s.position); err != nil {
		return nil, 0, err
	}
	s.position += int64(len(endBytes))
	if !bytes.Equal(endBytes, EndMagicNumber[:]) {
		s.Error("Wrong end magic number！", zap.ByteString("expect", EndMagicNumber[:]), zap.ByteString("actual", endBytes))
		return nil, 0, errors.New("Wrong end magic number！")
	}

	p := new(bytes.Buffer)
	p.Write(startBytes)
	p.Write(versionBytes)
	p.Write(dataLenBytes)
	p.Write(offsetBytes)
	p.Write(appliIndexBytes)
	p.Write(data)
	p.Write(endBytes)

	return p.Bytes(), Encoding.Uint64(appliIndexBytes), nil

}
