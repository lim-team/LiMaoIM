package db

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lim-team/LiMaoIM/pkg/keylock"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	bolt "go.etcd.io/bbolt"
)

// FileDB FileDB
type FileDB struct {
	dataDir           string
	pool              *Pool
	slotCount         int
	slotMap           sync.Map
	db                *bolt.DB
	lock              *keylock.KeyLock
	userTokenPrefix   string
	channelPrefix     string
	subscribersPrefix string
	denylistPrefix    string
	allowlistPrefix   string
	notifyQueuePrefix string
	scanner           *Scanner
}

// NewFileDB NewFileDB
func NewFileDB(dataDir string, segmentMaxBytes int64, slotCount int) *FileDB {
	f := &FileDB{
		dataDir:           dataDir,
		pool:              NewPool(dataDir, segmentMaxBytes),
		slotCount:         slotCount,
		scanner:           NewScanner(),
		slotMap:           sync.Map{},
		lock:              keylock.NewKeyLock(),
		userTokenPrefix:   "userToken:",
		channelPrefix:     "channel:",
		subscribersPrefix: "subscribers:",
		denylistPrefix:    "denylist:",
		allowlistPrefix:   "allowlist:",
		notifyQueuePrefix: "notifyQueue",
	}
	var err error
	f.db, err = bolt.Open(filepath.Join(dataDir, "db"), 0600, nil)
	if err != nil {
		panic(err)
	}
	f.lock.StartCleanLoop()

	err = f.db.Batch(func(t *bolt.Tx) error {
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

func (f *FileDB) topic(channelID string, channelType uint8) *Topic {
	slot := f.getSlot(channelID, channelType)
	return slot.GetTopic(f.topicName(channelID, channelType))
}

func (f *FileDB) getSlot(channelID string, channelType uint8) *Slot {
	return f.getOrNewSlot(f.slotNumForChannel(channelID, channelType))
}

func (f *FileDB) metaDataFileName() string {
	return path.Join(f.dataDir, "limao.meta.dat")
}

func (f *FileDB) topicName(channelID string, channelType uint8) string {
	return fmt.Sprintf("%d-%s", channelType, channelID)
}

func (f *FileDB) slotNumForChannel(channelID string, channelType uint8) uint {
	return GetSlotNum(f.slotCount, fmt.Sprintf("%s-%d", channelID, channelType))
}

func (f *FileDB) slotNum(key string) uint {
	return GetSlotNum(f.slotCount, key)
}

func (f *FileDB) getOrNewSlot(slotNum uint) *Slot {
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
	slot := GetSlotNum(f.slotCount, key)
	return f.getSlotBucket(slot, t)
}
func (f *FileDB) getSlotBucket(slotNum uint, t *bolt.Tx) (*bolt.Bucket, error) {
	return t.Bucket([]byte(fmt.Sprintf("%d", slotNum))), nil
}

func (f *FileDB) set(slot uint, key []byte, value []byte) error {
	err := f.db.Update(func(t *bolt.Tx) error {
		bucket, err := f.getSlotBucket(slot, t)
		if err != nil {
			return err
		}
		return bucket.Put(key, value)
	})
	return err
}

func (f *FileDB) get(slot uint, key []byte) ([]byte, error) {
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

func (f *FileDB) delete(slot uint, key []byte) error {
	return f.db.Update(func(t *bolt.Tx) error {
		bucket, err := f.getSlotBucket(slot, t)
		if err != nil {
			return err
		}
		return bucket.Delete(key)
	})
}

func (f *FileDB) addList(slotNum uint, key string, valueList []string) error {
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

func (f *FileDB) getList(slotNum uint, key string) ([]string, error) {
	f.lock.Lock(key)
	defer f.lock.Unlock(key)

	var values []string
	err := f.db.View(func(t *bolt.Tx) error {
		bucket, err := f.getSlotBucketWithKey(key, t)
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

func (f *FileDB) removeList(slotNum uint, key string, uids []string) error {
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
