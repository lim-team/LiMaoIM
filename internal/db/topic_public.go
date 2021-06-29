package db

import (
	"go.uber.org/zap"
)

// NextOffset NextOffset
func (t *Topic) NextOffset() (uint32, error) {
	return t.lastSeq.Inc(), nil
}

// AppendLog AppendLog
func (t *Topic) AppendLog(log ILog) (int, error) {
	activeSegment := t.getActiveSegment()

	if activeSegment.index.IsFull() || activeSegment.position >= t.segmentMaxBytes {
		activeSegment.Close()
		t.roll(log)
	}
	return t.getActiveSegment().AppendLog(log)
}

// ReadLogs ReadLogs
func (t *Topic) ReadLogs(offset int64, limit uint64, callback func(data []byte) error) error {
	baseOffset, err := t.baseOffset(offset)
	if err != nil {
		return err
	}

	readCount := 0
	var segment = t.getOrNewSegment(baseOffset)
	err = segment.ReadLogs(offset, limit, func(data []byte) error {
		readCount++
		if callback != nil {
			err := callback(data)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if readCount < int(limit) {
		nextBaseOffst := t.nextBaseOffset(baseOffset)
		if nextBaseOffst != -1 {
			nextSegment := t.getOrNewSegment(nextBaseOffst)
			err := nextSegment.ReadLogsAtPosition(0, limit-uint64(readCount), callback)
			if err != nil {
				t.Error("Failed to read the remaining logs", zap.Error(err), zap.Int64("position", 0), zap.Uint64("limit", limit-uint64(readCount)))
				return err
			}
		}
	}
	return nil

}

// ReadLogAt ReadLogAt
func (t *Topic) ReadLogAt(offset int64, log ILog) error {
	baseOffset, err := t.baseOffset(offset)
	if err != nil {
		return err
	}
	var segment = t.getOrNewSegment(baseOffset)
	return segment.ReadAt(offset, log)
}

// Sync Sync
func (t *Topic) Sync() error {
	return t.getActiveSegment().Sync()
}
