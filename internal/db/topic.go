package db

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	satomic "sync/atomic"

	lru "github.com/hashicorp/golang-lru"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// Topic Topic
type Topic struct {
	topic    string
	topicDir string
	limlog.Log
	segments        []int64
	activeSegment   satomic.Value
	segmentMaxBytes int64
	segmentCache    *lru.Cache
	sync.Mutex
	lastSeq    atomic.Uint32
	lastMemLog satomic.Value // 最新内存中的一条日志
}

// NewTopic NewTopic
func NewTopic(slotDir, topic string, segmentMaxBytes int64) *Topic {

	topicDir := filepath.Join(slotDir, "topics", topic)
	t := &Topic{
		topic:           topic,
		topicDir:        topicDir,
		Log:             limlog.NewLIMLog(fmt.Sprintf("Topic[%s]", topicDir)),
		segments:        make([]int64, 0),
		segmentMaxBytes: segmentMaxBytes,
	}
	cache, err := lru.NewWithEvict(1000, func(key, value interface{}) {
		sg := value.(*Segment)
		err := sg.Close()
		if err != nil {
			t.Warn("segment close fail", zap.Error(err))
		}
	})
	t.segmentCache = cache
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(t.topicDir, FileDefaultMode)
	if err != nil {
		t.Error("mkdir slot dir fail")
		panic(err)
	}
	t.initSegments()
	return t
}

func (t *Topic) initSegments() {

	files, err := ioutil.ReadDir(t.topicDir)
	if err != nil {
		t.Error("read dir fail!", zap.String("topicDir", t.topicDir))
		panic(err)
	}
	segments := make([]int64, 0)
	for _, file := range files {
		// if this file is an index file, make sure it has a corresponding .log file
		if strings.HasSuffix(file.Name(), indexSuffix) {
			_, err := os.Stat(filepath.Join(t.topicDir, strings.Replace(file.Name(), indexSuffix, logSuffix, 1)))
			if os.IsNotExist(err) {
				if err := os.Remove(file.Name()); err != nil {
					t.Error("remove index file fail!", zap.String("file", file.Name()))
					panic(err)
				}
			}
		} else if strings.HasSuffix(file.Name(), logSuffix) {
			offsetStr := strings.TrimSuffix(file.Name(), logSuffix)
			baseOffset, err := strconv.ParseInt(offsetStr, 10, 64)
			if err != nil {
				continue
			}
			segments = append(segments, baseOffset)
		}
	}

	t.segments = t.sortSegments(segments)
	if len(t.segments) == 0 {
		t.segments = append(t.segments, 0)
	}
	lastSegmentBaseOffset := t.segments[len(t.segments)-1]
	sg := NewSegment(t.topicDir, lastSegmentBaseOffset)
	lastMsgStartPosition, err := sg.SanityCheck()
	if err != nil {
		panic(err)
	}
	if lastMsgStartPosition == 0 {
		t.lastSeq.Store(uint32(lastSegmentBaseOffset))
	} else {
		offset, _, err := sg.readAtPosition(lastMsgStartPosition)
		if err != nil {
			panic(err)
		}
		t.lastSeq.Store(uint32(offset))
	}
	t.segmentCache.Add(lastSegmentBaseOffset, sg)
	t.activeSegment.Store(sg)

}

func (t *Topic) sortSegments(segments []int64) []int64 {
	for n := 0; n <= len(segments); n++ {
		for i := 1; i < len(segments)-n; i++ {
			if segments[i] < segments[i-1] {
				segments[i], segments[i-1] = segments[i-1], segments[i]
			}
		}
	}
	return segments
}

func (t *Topic) getOrNewSegment(baseOffset int64) *Segment {
	var segment *Segment
	sg, ok := t.segmentCache.Get(baseOffset)
	if ok {
		segment = sg.(*Segment)
	} else {
		segment = NewSegment(t.topicDir, baseOffset)
		t.segmentCache.Add(baseOffset, segment)
	}
	return segment
}

func (t *Topic) baseOffset(offset int64) (int64, error) {
	for i := 0; i < len(t.segments); i++ {
		baseOffst := t.segments[i]
		if i+1 < len(t.segments) {
			if offset >= baseOffst && offset <= t.segments[i+1] {
				return baseOffst, nil
			}
		} else {
			if offset >= baseOffst {
				return baseOffst, nil
			}
		}
	}
	return 0, errors.New("baseOffset not found")
}

func (t *Topic) nextBaseOffset(baseOffset int64) int64 {
	for i := 0; i < len(t.segments); i++ {
		baseOffst := t.segments[i]
		if baseOffst == baseOffset {
			if i+1 < len(t.segments) {
				return t.segments[i+1]
			}
			break
		}
	}
	return -1
}

func (t *Topic) roll(log ILog) {
	segment := NewSegment(t.topicDir, log.Offset())
	t.setActiveSegment(segment)

	t.Lock()
	t.segments = append(t.segments, log.Offset())
	t.Unlock()

}

// GetActiveSegment GetActiveSegment
func (t *Topic) getActiveSegment() *Segment {
	return t.activeSegment.Load().(*Segment)
}

func (t *Topic) setActiveSegment(sg *Segment) {
	t.activeSegment.Store(sg)
}

// Close Close
func (t *Topic) Close() error {

	v := t.activeSegment.Load()
	if v != nil {
		return v.(*Segment).Close()
	}
	keys := t.segmentCache.Keys()
	for _, key := range keys {
		t.segmentCache.Remove(key)
	}
	return nil
}
