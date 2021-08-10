package db

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lim-team/LiMaoIM/pkg/keylock"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"github.com/tangtaoit/limnet/pkg/limlog"
	bolt "go.etcd.io/bbolt"
)

// FileDB FileDB
type FileDB struct {
	dataDir                   string
	pool                      *Pool
	slotCount                 int
	slotMap                   sync.Map
	db                        *bolt.DB
	lock                      *keylock.KeyLock
	userTokenPrefix           string
	channelPrefix             string
	subscribersPrefix         string
	denylistPrefix            string
	allowlistPrefix           string
	notifyQueuePrefix         string
	userSeqPrefix             string
	nodeInFlightDataPrefix    string
	scanner                   *Scanner
	rootBucketPrefix          string
	messageOfUserCursorPrefix string
	limlog.Log
}

// NewFileDB NewFileDB
func NewFileDB(dataDir string, segmentMaxBytes int64, slotCount int) *FileDB {
	f := &FileDB{
		dataDir:                   dataDir,
		pool:                      NewPool(dataDir, segmentMaxBytes),
		slotCount:                 slotCount,
		scanner:                   NewScanner(),
		slotMap:                   sync.Map{},
		lock:                      keylock.NewKeyLock(),
		userTokenPrefix:           "userToken:",
		channelPrefix:             "channel:",
		subscribersPrefix:         "subscribers:",
		denylistPrefix:            "denylist:",
		allowlistPrefix:           "allowlist:",
		notifyQueuePrefix:         "notifyQueue",
		userSeqPrefix:             "userSeq:",
		nodeInFlightDataPrefix:    "nodeInFlightData",
		rootBucketPrefix:          "limaoRoot",
		messageOfUserCursorPrefix: "messageOfUserCursor:",
		Log:                       limlog.NewLIMLog("FileDB"),
	}
	var err error
	f.db, err = bolt.Open(filepath.Join(dataDir, "limaoim.db"), 0600, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		panic(err)
	}
	f.lock.StartCleanLoop()

	err = f.db.Batch(func(t *bolt.Tx) error {
		_, err = t.CreateBucketIfNotExists([]byte(f.rootBucketPrefix))
		if err != nil {
			return err
		}
		_, err = t.CreateBucketIfNotExists([]byte(f.notifyQueuePrefix))
		if err != nil {
			return err
		}
		for i := 0; i < slotCount; i++ {
			_, err := t.CreateBucketIfNotExists([]byte(fmt.Sprintf("%d", i)))
			if err != nil {
				panic(err)
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	return f
}

func (f *FileDB) getRootBucket(t *bolt.Tx) *bolt.Bucket {
	return t.Bucket([]byte(f.rootBucketPrefix))
}

func (f *FileDB) topic(channelID string, channelType uint8) *Topic {
	slot := f.getSlot(channelID, channelType)
	return slot.GetTopic(f.topicName(channelID, channelType))
}

func (f *FileDB) getSlot(channelID string, channelType uint8) *Slot {
	return f.getOrNewSlot(f.slotNumForChannel(channelID, channelType))
}

func (f *FileDB) topicName(channelID string, channelType uint8) string {
	return fmt.Sprintf("%d-%s", channelType, channelID)
}

func (f *FileDB) slotNumForChannel(channelID string, channelType uint8) uint32 {
	return util.GetSlotNum(f.slotCount, channelID)
}

func (f *FileDB) slotNum(key string) uint32 {
	return util.GetSlotNum(f.slotCount, key)
}

func (f *FileDB) getOrNewSlot(slotNum uint32) *Slot {
	f.lock.Lock(fmt.Sprintf("slotNum-%d", slotNum))
	defer f.lock.Unlock(fmt.Sprintf("slotNum-%d", slotNum))
	v, ok := f.slotMap.Load(slotNum)
	var slot *Slot
	if ok {
		slot = v.(*Slot)
	} else {
		slot = f.pool.GetSlot(slotNum)
		f.slotMap.Store(slotNum, slot)
	}
	return slot
}

func (f *FileDB) getSlotBucketWithKey(key string, t *bolt.Tx) (*bolt.Bucket, error) {
	slot := util.GetSlotNum(f.slotCount, key)
	return f.getSlotBucket(slot, t)
}
func (f *FileDB) getSlotBucket(slotNum uint32, t *bolt.Tx) (*bolt.Bucket, error) {
	return t.Bucket([]byte(fmt.Sprintf("%d", slotNum))), nil
}

func (f *FileDB) set(slot uint32, key []byte, value []byte) error {
	err := f.db.Update(func(t *bolt.Tx) error {
		bucket, err := f.getSlotBucket(slot, t)
		if err != nil {
			return err
		}
		return bucket.Put(key, value)
	})
	return err
}

func (f *FileDB) get(slot uint32, key []byte) ([]byte, error) {
	var value []byte
	err := f.db.View(func(t *bolt.Tx) error {
		bucket, err := f.getSlotBucket(slot, t)
		if err != nil {
			return err
		}
		value = bucket.Get(key)
		return nil
	})
	return value, err
}

func (f *FileDB) delete(slot uint32, key []byte) error {
	return f.db.Update(func(t *bolt.Tx) error {
		bucket, err := f.getSlotBucket(slot, t)
		if err != nil {
			return err
		}
		return bucket.Delete(key)
	})
}

func (f *FileDB) addList(slotNum uint32, key string, valueList []string) error {
	f.lock.Lock(key)
	defer f.lock.Unlock(key)

	err := f.db.Update(func(t *bolt.Tx) error {
		bucket, err := f.getSlotBucket(slotNum, t)
		if err != nil {
			return err
		}
		value := bucket.Get([]byte(key))
		list := make([]string, 0)
		if len(value) > 0 {
			values := strings.Split(string(value), ",")
			if len(values) > 0 {
				list = append(list, values...)
			}
		}
		list = append(list, valueList...)
		return bucket.Put([]byte(key), []byte(strings.Join(list, ",")))
	})
	return err
}

func (f *FileDB) getList(slotNum uint32, key string) ([]string, error) {
	f.lock.Lock(key)
	defer f.lock.Unlock(key)

	var values []string
	err := f.db.View(func(t *bolt.Tx) error {
		bucket, err := f.getSlotBucket(slotNum, t)
		if err != nil {
			return err
		}
		value := bucket.Get([]byte(key))
		if len(value) > 0 {
			values = strings.Split(string(value), ",")
			return nil
		}
		return nil
	})
	return values, err
}

func (f *FileDB) removeList(slotNum uint32, key string, uids []string) error {
	f.lock.Lock(key)
	defer f.lock.Unlock(key)
	err := f.db.Update(func(t *bolt.Tx) error {
		bucket, err := f.getSlotBucketWithKey(key, t)
		if err != nil {
			return err
		}
		value := bucket.Get([]byte(key))
		list := make([]string, 0)
		if len(value) > 0 {
			values := strings.Split(string(value), ",")
			if len(values) > 0 {
				for _, v := range values {
					var has = false
					for _, uid := range uids {
						if v == uid {
							has = true
							break
						}
					}
					if !has {
						list = append(list, v)
					}
				}
			}
		}
		return bucket.Put([]byte(key), []byte(strings.Join(list, ",")))
	})
	return err

}

func (f *FileDB) getUserTokenKey(uid string, deviceFlag lmproto.DeviceFlag) string {
	return fmt.Sprintf("%s%s-%d", f.userTokenPrefix, uid, deviceFlag.ToUint8())
}
func (f *FileDB) getChannelKey(channelID string, channelType uint8) string {
	return fmt.Sprintf("%s%s-%d", f.channelPrefix, channelID, channelType)
}

func (f *FileDB) getSubscribersKey(channelID string, channelType uint8) string {
	return fmt.Sprintf("%s%s-%d", f.subscribersPrefix, channelID, channelType)
}

func (f *FileDB) getDenylistKey(channelID string, channelType uint8) string {
	return fmt.Sprintf("%s%s-%d", f.denylistPrefix, channelID, channelType)
}
func (f *FileDB) getAllowlistKey(channelID string, channelType uint8) string {
	return fmt.Sprintf("%s%s-%d", f.allowlistPrefix, channelID, channelType)
}
func (f *FileDB) getAppliIndexKey() string {
	return fmt.Sprintf("appliIndex")
}

func (f *FileDB) getMessageOfUserCursorKey(uid string) string {
	return fmt.Sprintf("%s%s", f.messageOfUserCursorPrefix, uid)
}

func (f *FileDB) getSubdir(dir string) ([]string, error) {
	fd, err := os.Open(dir)
	if err != nil {
		if !os.IsExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer fd.Close()
	subdirs, err := fd.Readdir(-1)
	if err != nil {
		return nil, err
	}
	dirs := make([]string, 0, len(subdirs))
	for _, subdir := range subdirs {
		if subdir.IsDir() {
			dirs = append(dirs, subdir.Name())
		}

	}
	return dirs, nil
}
func (f *FileDB) getSubfiles(dir string, suffix string) ([]string, error) {
	fd, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	subdirs, err := fd.Readdir(-1)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(subdirs))
	for _, subdir := range subdirs {
		if !subdir.IsDir() && strings.HasSuffix(subdir.Name(), suffix) {
			files = append(files, subdir.Name())
		}

	}
	return files, nil
}

func (f *FileDB) getSegmentInfo(slots []uint32) ([]*segmentInfo, error) {
	if len(slots) == 0 {
		return nil, nil
	}
	slotDir := filepath.Join(f.dataDir, "slots")
	slotList := make([]string, 0, len(slots))
	for _, s := range slots {
		slotList = append(slotList, fmt.Sprintf("%d", s))
	}
	segmentInfos := make([]*segmentInfo, 0)
	for _, slotStr := range slotList {
		segmentI := &segmentInfo{}
		topicDir := filepath.Join(slotDir, slotStr, "topics")
		topicList, err := f.getSubdir(topicDir)
		if err != nil {
			return nil, err
		}
		slot, err := strconv.ParseUint(slotStr, 10, 64)
		if err != nil {
			return nil, err
		}
		segmentI.slot = uint32(slot)
		for _, topic := range topicList {
			segmentI.topic = topic
			logFiles, err := f.getSubfiles(filepath.Join(topicDir, topic, "logs"), logSuffix)
			if err != nil {
				return nil, err
			}
			sort.Sort(sort.StringSlice(logFiles))
			baseOffsets := make([]int64, 0, len(logFiles))
			for _, logFile := range logFiles {
				baseOffset, err := strconv.ParseInt(strings.TrimSuffix(logFile, logSuffix), 10, 64)
				if err != nil {
					return nil, err
				}
				baseOffsets = append(baseOffsets, baseOffset)
			}
			segmentI.baseOffsets = baseOffsets
		}
		segmentInfos = append(segmentInfos, segmentI)
	}
	return segmentInfos, nil
}

func (f *FileDB) getAllSegmentInfo() ([]*segmentInfo, error) {
	slotDir := filepath.Join(f.dataDir, "slots")
	slotList, err := f.getSubdir(slotDir)
	if err != nil {
		return nil, err
	}
	segmentInfos := make([]*segmentInfo, 0)
	for _, slotStr := range slotList {
		segmentI := &segmentInfo{}
		topicDir := filepath.Join(slotDir, slotStr, "topics")
		topicList, err := f.getSubdir(topicDir)
		if err != nil {
			return nil, err
		}
		slot, err := strconv.ParseUint(slotStr, 10, 64)
		if err != nil {
			return nil, err
		}
		segmentI.slot = uint32(slot)
		for _, topic := range topicList {
			segmentI.topic = topic
			logFiles, err := f.getSubfiles(filepath.Join(topicDir, topic, "logs"), logSuffix)
			if err != nil {
				return nil, err
			}
			sort.Sort(sort.StringSlice(logFiles))
			baseOffsets := make([]int64, 0, len(logFiles))
			for _, logFile := range logFiles {
				baseOffset, err := strconv.ParseInt(strings.TrimSuffix(logFile, logSuffix), 10, 64)
				if err != nil {
					return nil, err
				}
				baseOffsets = append(baseOffsets, baseOffset)
			}
			segmentI.baseOffsets = baseOffsets
		}
		segmentInfos = append(segmentInfos, segmentI)
	}
	return segmentInfos, nil
}

type segmentInfo struct {
	slot        uint32
	topic       string
	baseOffsets []int64
}

func newSegmentInfo(slot uint32, topic string, baseOffsets []int64) *segmentInfo {
	return &segmentInfo{
		slot:        slot,
		topic:       topic,
		baseOffsets: baseOffsets,
	}
}

func (f *FileDB) writeSnapshotHeader(slot int, topic string, baseOffset int64, w io.Writer) error {
	slotByte := make([]byte, 4)
	Encoding.PutUint32(slotByte, uint32(slot))
	if _, err := w.Write(slotByte); err != nil {
		return err
	}
	topicLenBytes := make([]byte, 2)
	Encoding.PutUint16(topicLenBytes, uint16(len(topic)))
	if _, err := w.Write([]byte(topic)); err != nil {
		return err
	}
	baseOffsetBytes := make([]byte, 8)
	Encoding.PutUint64(baseOffsetBytes, uint64(baseOffset))
	if _, err := w.Write(baseOffsetBytes); err != nil {
		return err
	}
	return nil
}
