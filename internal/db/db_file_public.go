package db

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"go.etcd.io/bbolt"
	bolt "go.etcd.io/bbolt"
)

var (
	// ErrorTokenNotFound ErrorTokenNotFound
	ErrorTokenNotFound = errors.New("token not found")
)

// Open Open
func (f *FileDB) Open() error {
	return nil
}

// Close Close
func (f *FileDB) Close() error {
	f.lock.StopCleanLoop()
	err := f.db.Close()
	if err != nil {
		return err
	}
	return f.pool.Close()
}

// Sync Sync
func (f *FileDB) Sync() error {
	return f.pool.Sync()
}

// UpdateUserToken UpdateUserToken
func (f *FileDB) UpdateUserToken(uid string, deviceFlag lmproto.DeviceFlag, deviceLevel lmproto.DeviceLevel, token string) error {
	slotNum := f.slotNum(uid)
	return f.set(slotNum, []byte(f.getUserTokenKey(uid, deviceFlag)), []byte(util.ToJSON(map[string]string{
		"device_level": fmt.Sprintf("%d", deviceLevel),
		"token":        token,
	})))
}

// GetUserToken GetUserToken
func (f *FileDB) GetUserToken(uid string, deviceFlag lmproto.DeviceFlag) (string, lmproto.DeviceLevel, error) {
	slotNum := f.slotNum(uid)
	value, err := f.get(slotNum, []byte(f.getUserTokenKey(uid, deviceFlag)))
	if err != nil {
		return "", 0, err
	}
	if len(value) == 0 {
		return "", 0, nil
	}
	var resultMap map[string]string
	err = util.ReadJSONByByte(value, &resultMap)
	if err != nil {
		return "", 0, err
	}
	token := resultMap["token"]
	level, _ := strconv.Atoi(resultMap["device_level"])
	return token, lmproto.DeviceLevel(level), nil
}

// AddOrUpdateChannel AddOrUpdateChannel
func (f *FileDB) AddOrUpdateChannel(channelID string, channelType uint8, data map[string]interface{}) error {
	slotNum := f.slotNumForChannel(channelID, channelType)
	return f.set(slotNum, []byte(f.getChannelKey(channelID, channelType)), []byte(util.ToJSON(data)))
}

// GetChannel GetChannel
func (f *FileDB) GetChannel(channelID string, channelType uint8) (map[string]interface{}, error) {
	slotNum := f.slotNumForChannel(channelID, channelType)
	value, err := f.get(slotNum, []byte(f.getChannelKey(channelID, channelType)))
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	var data map[string]interface{}
	err = util.ReadJSONByByte(value, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// DeleteChannel DeleteChannel
func (f *FileDB) DeleteChannel(channelID string, channelType uint8) error {
	slotNum := f.slotNumForChannel(channelID, channelType)
	return f.delete(slotNum, []byte(f.getChannelKey(channelID, channelType)))
}

// ExistChannel ExistChannel
func (f *FileDB) ExistChannel(channelID string, channelType uint8) (bool, error) {
	value, err := f.GetChannel(channelID, channelType)
	if err != nil {
		return false, err
	}
	if value != nil {
		return true, nil
	}
	return false, nil
}

// AddSubscribers AddSubscribers
func (f *FileDB) AddSubscribers(channelID string, channelType uint8, uids []string) error {
	key := f.getSubscribersKey(channelID, channelType)
	slotNum := f.slotNumForChannel(channelID, channelType)
	return f.addList(slotNum, key, uids)
}

// GetSubscribers GetSubscribers
func (f *FileDB) GetSubscribers(channelID string, channelType uint8) ([]string, error) {
	key := f.getSubscribersKey(channelID, channelType)
	slotNum := f.slotNumForChannel(channelID, channelType)
	return f.getList(slotNum, key)
}

// RemoveSubscribers RemoveSubscribers
func (f *FileDB) RemoveSubscribers(channelID string, channelType uint8, uids []string) error {
	key := f.getSubscribersKey(channelID, channelType)
	slotNum := f.slotNumForChannel(channelID, channelType)
	return f.removeList(slotNum, key, uids)
}

// RemoveAllSubscriber RemoveAllSubscriber
func (f *FileDB) RemoveAllSubscriber(channelID string, channelType uint8) error {
	key := f.getSubscribersKey(channelID, channelType)
	slotNum := f.slotNumForChannel(channelID, channelType)
	err := f.db.Update(func(t *bolt.Tx) error {
		bucket, err := f.getSlotBucket(slotNum, t)
		if err != nil {
			return err
		}
		return bucket.Delete([]byte(key))
	})
	return err
}

// AddDenylist AddDenylist
func (f *FileDB) AddDenylist(channelID string, channelType uint8, uids []string) error {
	key := f.getDenylistKey(channelID, channelType)
	slotNum := f.slotNumForChannel(channelID, channelType)
	return f.addList(slotNum, key, uids)
}

// GetDenylist GetDenylist
func (f *FileDB) GetDenylist(channelID string, channelType uint8) ([]string, error) {
	key := f.getDenylistKey(channelID, channelType)
	slotNum := f.slotNumForChannel(channelID, channelType)
	return f.getList(slotNum, key)
}

// RemoveDenylist RemoveDenylist
func (f *FileDB) RemoveDenylist(channelID string, channelType uint8, uids []string) error {
	key := f.getDenylistKey(channelID, channelType)
	slotNum := f.slotNumForChannel(channelID, channelType)
	return f.removeList(slotNum, key, uids)
}

// RemoveAllDenylist RemoveAllDenylist
func (f *FileDB) RemoveAllDenylist(channelID string, channelType uint8) error {
	key := f.getDenylistKey(channelID, channelType)
	f.lock.Lock(key)
	defer f.lock.Unlock(key)
	slotNum := f.slotNumForChannel(channelID, channelType)
	return f.delete(slotNum, []byte(key))
}

// GetAllowlist GetAllowlist
func (f *FileDB) GetAllowlist(channelID string, channelType uint8) ([]string, error) {
	key := f.getAllowlistKey(channelID, channelType)
	slotNum := f.slotNumForChannel(channelID, channelType)
	return f.getList(slotNum, key)
}

// AddAllowlist AddAllowlist
func (f *FileDB) AddAllowlist(channelID string, channelType uint8, uids []string) error {
	key := f.getAllowlistKey(channelID, channelType)
	slotNum := f.slotNumForChannel(channelID, channelType)
	return f.addList(slotNum, key, uids)
}

// RemoveAllowlist RemoveAllowlist
func (f *FileDB) RemoveAllowlist(channelID string, channelType uint8, uids []string) error {
	key := f.getAllowlistKey(channelID, channelType)
	slotNum := f.slotNumForChannel(channelID, channelType)
	return f.removeList(slotNum, key, uids)
}

// RemoveAllAllowlist RemoveAllAllowlist
func (f *FileDB) RemoveAllAllowlist(channelID string, channelType uint8) error {
	key := f.getAllowlistKey(channelID, channelType)
	f.lock.Lock(key)
	defer f.lock.Unlock(key)
	slotNum := f.slotNumForChannel(channelID, channelType)
	return f.delete(slotNum, []byte(key))
}

// GetNextMessageSeq GetNextMessageSeq
func (f *FileDB) GetNextMessageSeq(channelID string, channelType uint8) (uint32, error) {
	return f.topic(channelID, channelType).NextOffset()
}

// GetUserNextMessageSeq GetUserNextMessageSeq
func (f *FileDB) GetUserNextMessageSeq(uid string) (uint32, error) {
	slot := f.slotNum(uid)
	return f.getOrNewSlot(slot).GetTopic(uid).NextOffset()
}

// AppendMessage AppendMessage
func (f *FileDB) AppendMessage(m *Message) (int, error) {
	return f.topic(m.ChannelID, m.ChannelType).AppendLog(m)
}

// AppendMessageOfUser Append message to user
func (f *FileDB) AppendMessageOfUser(uid string, m *Message) (int, error) {
	slot := f.slotNum(uid)
	return f.getOrNewSlot(slot).GetTopic(uid).AppendLog(m)
}

// AppendMessageOfNotifyQueue AppendMessageOfNotifyQueue
func (f *FileDB) AppendMessageOfNotifyQueue(m *Message) error {
	return f.db.Update(func(t *bolt.Tx) error {
		bucket := t.Bucket([]byte(f.notifyQueuePrefix))
		seq, err := bucket.NextSequence()
		if err != nil {
			return err
		}
		return bucket.Put(itob(seq), []byte(util.ToJSON(m)))
	})
}

// GetMessagesOfNotifyQueue GetMessagesOfNotifyQueue
func (f *FileDB) GetMessagesOfNotifyQueue(count int) ([]*Message, error) {
	messages := make([]*Message, 0)
	err := f.db.View(func(t *bolt.Tx) error {
		bucket := t.Bucket([]byte(f.notifyQueuePrefix))
		c := bucket.Cursor()
		i := 0
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if i > count-1 {
				break
			}
			var message *Message
			err := util.ReadJSONByByte(v, &message)
			if err != nil {
				return err
			}
			messages = append(messages, message)
			i++
		}
		return nil
	})
	return messages, err
}

// RemoveMessagesOfNotifyQueue RemoveMessagesOfNotifyQueue
func (f *FileDB) RemoveMessagesOfNotifyQueue(messageIDs []int64) error {
	if len(messageIDs) == 0 {
		return nil
	}
	return f.db.Update(func(t *bolt.Tx) error {
		bucket := t.Bucket([]byte(f.notifyQueuePrefix))
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var message *Message
			err := util.ReadJSONByByte(v, &message)
			if err != nil {
				return err
			}
			for _, messageID := range messageIDs {
				if message.MessageID == messageID {
					err = bucket.Delete(k)
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

// AddOrUpdateConversations AddOrUpdateConversations
func (f *FileDB) AddOrUpdateConversations(uid string, conversations []*Conversation) error {
	key := fmt.Sprintf("conversation:%s", uid)
	return f.db.Update(func(t *bbolt.Tx) error {
		bucket, err := f.getSlotBucketWithKey(uid, t)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(key), []byte(util.ToJSON(conversations)))
	})
}

// GetConversations GetConversations
func (f *FileDB) GetConversations(uid string) ([]*Conversation, error) {
	key := fmt.Sprintf("conversation:%s", uid)
	var conversations []*Conversation
	err := f.db.View(func(t *bbolt.Tx) error {
		bucket, err := f.getSlotBucketWithKey(uid, t)
		if err != nil {
			return err
		}
		value := bucket.Get([]byte(key))
		if len(value) > 0 {
			err = util.ReadJSONByByte(value, &conversations)
			return err
		}
		return nil

	})
	return conversations, err
}

// GetMessages 获取消息
func (f *FileDB) GetMessages(channelID string, channelType uint8, offset uint32, limit uint64) ([]*Message, error) {
	var messages = make([]*Message, 0, limit)
	err := f.topic(channelID, channelType).ReadLogs(int64(offset), limit, func(data []byte) error {
		m := &Message{}
		err := UnmarshalMessage(data, m)
		if err != nil {
			return err
		}
		messages = append(messages, m)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// GetMessage GetMessage
func (f *FileDB) GetMessage(channelID string, channelType uint8, messageSeq uint32) (*Message, error) {
	var message = &Message{}
	err := f.topic(channelID, channelType).ReadLogAt(int64(messageSeq), message)
	return message, err
}

// GetMetaData GetMetaData
func (f *FileDB) GetMetaData() (uint64, error) {
	var applied uint64 = 0
	f.db.View(func(t *bbolt.Tx) error {
		bucket, err := t.CreateBucketIfNotExists([]byte("metadata"))
		if err != nil {
			return err
		}
		value := bucket.Get([]byte(f.getAppliIndexKey()))
		if len(value) > 0 {
			applied, _ = strconv.ParseUint(string(value), 10, 64)
		}
		return nil
	})

	return applied, nil
}

// SaveMetaData SaveMetaData
func (f *FileDB) SaveMetaData(appliIndex uint64) error {
	return f.db.Update(func(t *bbolt.Tx) error {
		bucket, err := t.CreateBucketIfNotExists([]byte("metadata"))
		if err != nil {
			return err
		}
		return bucket.Put([]byte(f.getAppliIndexKey()), []byte(fmt.Sprintf("%d", appliIndex)))
	})
}

// PrepareSnapshot PrepareSnapshot
func (f *FileDB) PrepareSnapshot() (*Snapshot, error) {
	appliIndex, err := f.GetMetaData()
	if err != nil {
		return nil, err
	}
	return NewSnapshot(appliIndex), nil
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

// SaveSnapshot SaveSnapshot
func (f *FileDB) SaveSnapshot(snapshot *Snapshot, w io.Writer) error {

	w.Write(SnapshotMagicNumber[:])

	err := f.writeSnapshotHeader(1, "test", 1, w)
	if err != nil {
		return err
	}
	segmentInfos, err := f.getAllSegmentInfo()
	if err != nil {
		return err
	}
	if len(segmentInfos) <= 0 {
		return nil
	}
	for _, segmentInfo := range segmentInfos {
		for _, baseOffset := range segmentInfo.baseOffsets {
			for {
				segmentPath := filepath.Join(f.dataDir, "slots", fmt.Sprintf("%d", segmentInfo.slot), "topics", segmentInfo.topic, "logs", fmt.Sprintf(fileFormat, baseOffset, logSuffix))
				has, err := f.scanner.Scan(segmentPath, snapshot.AppliIndex, func(data []byte) error {
					_, err := w.Write(data)
					return err
				})
				if err != nil {
					return err
				}
				if !has {
					break
				}
			}
		}
	}
	_, err = w.Write(EndSnapshotMagicNumber[:])
	return err
}

func (f *FileDB) getSubdir(dir string) ([]string, error) {
	fd, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	subdirs, err := fd.ReadDir(-1)
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
	subdirs, err := fd.ReadDir(-1)
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
		slot, err := strconv.Atoi(slotStr)
		if err != nil {
			return nil, err
		}
		segmentI.slot = slot
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
	slot        int
	topic       string
	baseOffsets []int64
}

func newSegmentInfo(slot int, topic string, baseOffsets []int64) *segmentInfo {
	return &segmentInfo{
		slot:        slot,
		topic:       topic,
		baseOffsets: baseOffsets,
	}
}
