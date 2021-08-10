package db

import (
	"os"

	"go.uber.org/zap"
)

// NextOffset NextOffset
func (t *Topic) NextOffset() (uint32, error) {
	return t.lastSeq.Inc(), nil
}

func (t *Topic) GetLastSeq() (uint32, error) {
	return t.lastSeq.Load(), nil
}

// AppendLog AppendLog
func (t *Topic) AppendLog(log ILog) (int, error) {
	activeSegment := t.getActiveSegment()
	lastLog := t.lastMemLog.Load()
	if lastLog != nil && lastLog.(ILog).Offset() >= log.Offset() { // 追加的日志已经存在，直接返回啥也不干
		t.Warn("已经存在消息，不进行追加", zap.Int64("offset", log.Offset()))
		return 0, nil
	}
	t.lastMemLog.Store(log)
	if t.lastSeq.Load() < uint32(log.Offset()) {
		t.lastSeq.Store(uint32(log.Offset()))
	}
	if activeSegment.index.IsFull() || activeSegment.position >= t.segmentMaxBytes {
		t.roll(log)
	}
	return t.getActiveSegment().AppendLog(log)
}

// ReadLastLogs 获取最新的数量的日志
func (t *Topic) ReadLastLogs(limit uint64, callback func(data []byte) error) error {
	actLimit := limit
	var actOffset uint64 = 0
	if uint64(t.lastSeq.Load()) < limit {
		actLimit = uint64(t.lastSeq.Load()) + 1 // +1表示包含最后一条lastSeq的消息
		actOffset = 0
	} else {
		actLimit = limit
		actOffset = uint64(t.lastSeq.Load()) - limit + 1 // +1表示包含最后一条lastSeq的消息
	}
	return t.ReadLogs(int64(actOffset), actLimit, callback)
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
	nextBaseOffset := baseOffset
	for readCount < int(limit) {
		nextBaseOffset = t.nextBaseOffset(nextBaseOffset)
		if nextBaseOffset != -1 {
			nextSegment := t.getOrNewSegment(nextBaseOffset)
			err := nextSegment.readLogsAtPosition(0, limit-uint64(readCount), func(data []byte) error {
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
				t.Error("Failed to read the remaining logs", zap.Error(err), zap.Int64("position", 0), zap.Uint64("limit", limit-uint64(readCount)))
				return err
			}
		} else {
			break
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

// Delete Delete
func (t *Topic) Delete() error {
	return os.RemoveAll(t.topicDir)
}

// Sync Sync
func (t *Topic) Sync() error {
	return t.getActiveSegment().Sync()
}
