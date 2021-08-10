package db

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"go.etcd.io/bbolt"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"
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

	err := f.db.Close()
	if err != nil {
		return err
	}
	err = f.pool.Close()
	if err != nil {
		return err
	}

	f.lock.StopCleanLoop()

	return nil
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
	err := f.delete(slotNum, []byte(f.getChannelKey(channelID, channelType)))
	if err != nil {
		return err
	}
	return nil
}

// DeleteChannelAndClearMessages DeleteChannelAndClearMessages
func (f *FileDB) DeleteChannelAndClearMessages(channelID string, channelType uint8) error {
	err := f.DeleteChannel(channelID, channelType)
	if err != nil {
		return err
	}
	return f.DeleteMessages(channelID, channelType)
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
	// var seq uint64
	// err := f.db.Update(func(t *bolt.Tx) error {
	// 	b, err := f.getSlotBucket(slot, t)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	userBucket, err := b.CreateBucketIfNotExists([]byte(fmt.Sprintf("%s%s", f.userSeqPrefix, uid)))
	// 	if err != nil {
	// 		return err
	// 	}
	// 	seq, err = userBucket.NextSequence()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return nil
	// })
	// return uint32(seq), err
}

// AppendMessage AppendMessage
func (f *FileDB) AppendMessage(m *Message) (int, error) {
	slot := f.slotNumForChannel(m.ChannelID, m.ChannelType)
	topic := f.topicName(m.ChannelID, m.ChannelType)
	return f.appendMessage(slot, topic, m)
}

// AppendMessageOfUser Append message to user
func (f *FileDB) AppendMessageOfUser(m *Message) (int, error) {
	if strings.TrimSpace(m.QueueUID) == "" {
		return 0, errors.New("QueueUID不能为空！")
	}
	uid := m.QueueUID
	slot := f.slotNum(uid)
	return f.appendMessage(slot, uid, m)
}

func (f *FileDB) appendMessage(slot uint32, topic string, m *Message) (int, error) {
	return f.getOrNewSlot(slot).GetTopic(topic).AppendLog(m)
}

// UpdateMessageOfUserCursorIfNeed UpdateMessageOfUserCursorIfNeed
func (f *FileDB) UpdateMessageOfUserCursorIfNeed(uid string, offset uint32) error {
	slot := f.slotNum(uid)
	lastSeq, _ := f.getOrNewSlot(slot).GetTopic(uid).GetLastSeq()
	actOffset := offset
	if offset > lastSeq { // 如果传过来的大于系统里最新的 则用最新的
		actOffset = lastSeq
	}
	return f.db.Update(func(t *bolt.Tx) error {
		b, err := f.getSlotBucket(slot, t)
		if err != nil {
			return err
		}
		key := f.getMessageOfUserCursorKey(uid)
		value := b.Get([]byte(key))
		if len(value) > 0 {
			offset64, _ := strconv.ParseUint(string(value), 10, 64)
			oldOffset := uint32(offset64)
			if actOffset <= oldOffset && oldOffset < lastSeq {
				return nil
			}
		}
		return b.Put([]byte(key), []byte(fmt.Sprintf("%d", actOffset)))
	})
}

// GetMessageOfUserCursor GetMessageOfUserCursor
func (f *FileDB) GetMessageOfUserCursor(uid string) (uint32, error) {
	slot := f.slotNum(uid)
	var offset uint32 = 0
	err := f.db.View(func(t *bolt.Tx) error {
		b, err := f.getSlotBucket(slot, t)
		if err != nil {
			return err
		}
		value := b.Get([]byte(f.getMessageOfUserCursorKey(uid)))
		if len(value) > 0 {
			offset64, _ := strconv.ParseUint(string(value), 10, 64)
			offset = uint32(offset64)
		}
		return nil
	})
	return offset, err
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
	f.Info("GetMessages", zap.Uint32("offset", offset), zap.Uint64("limit", limit))
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
	f.Info("Reuslt-GetMessages", zap.Int("count", len(messages)))
	if len(messages) > 0 {
		f.Info("firstMessageSeq", zap.Uint32("messageSeq", messages[0].MessageSeq))
	}
	return messages, nil
}

// GetLastMessages 获取最新的消息
func (f *FileDB) GetLastMessages(channelID string, channelType uint8, endOffset uint32, limit uint64) ([]*Message, error) {
	var messages = make([]*Message, 0, limit)
	err := f.topic(channelID, channelType).ReadLastLogs(limit, func(data []byte) error {
		m := &Message{}
		err := UnmarshalMessage(data, m)
		if err != nil {
			return err
		}
		if endOffset != 0 && m.MessageSeq <= endOffset {
			return nil
		}
		messages = append(messages, m)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// GetMessagesOfUser 获取用户队列内的消息
func (f *FileDB) GetMessagesOfUser(uid string, offset uint32, limit uint64) ([]*Message, error) {
	slot := f.slotNum(uid)
	var messages = make([]*Message, 0, limit)

	maxOffset, err := f.GetMessageOfUserCursor(uid)
	if err != nil {
		return nil, err
	}
	offst := offset
	if offset < maxOffset {
		offst = maxOffset
	}

	fmt.Println("maxOffset-->", uid, offst)
	f.getOrNewSlot(slot).GetTopic(uid).ReadLogs(int64(offst+1), limit, func(data []byte) error {
		m := &Message{}
		err := UnmarshalMessage(data, m)
		if err != nil {
			return err
		}
		messages = append(messages, m)
		return nil
	})
	return messages, nil
}

// GetMessage GetMessage
func (f *FileDB) GetMessage(channelID string, channelType uint8, messageSeq uint32) (*Message, error) {
	var message = &Message{}
	err := f.topic(channelID, channelType).ReadLogAt(int64(messageSeq), message)
	return message, err
}

// DeleteMessages DeleteMessages
func (f *FileDB) DeleteMessages(channelID string, channelType uint8) error {
	f.topic(channelID, channelType)
	return nil
}

// GetMetaData GetMetaData
func (f *FileDB) GetMetaData() (uint64, error) {
	var applied uint64 = 0
	err := f.db.View(func(t *bbolt.Tx) error {
		bucket := f.getRootBucket(t)
		value := bucket.Get([]byte(f.getAppliIndexKey()))
		if len(value) > 0 {
			applied, _ = strconv.ParseUint(string(value), 10, 64)
		}
		return nil
	})
	return applied, err
}

// SaveMetaData SaveMetaData
func (f *FileDB) SaveMetaData(appliIndex uint64) error {
	return f.db.Update(func(t *bbolt.Tx) error {
		bucket := f.getRootBucket(t)
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

// BackupSlots BackupSlots
func (f *FileDB) BackupSlots(slots []byte, w io.Writer) error {

	slotBitM := util.NewSlotBitMapWithBits(slots)
	vauldSlots := slotBitM.GetVaildSlots()
	if len(vauldSlots) == 0 {
		return nil
	}
	var err error
	if _, err = w.Write(BackupMagicNumber[:]); err != nil {
		return err
	}
	if err = lmproto.WriteBinary(slots, w); err != nil {
		return err
	}

	if err = f.backupBaseData(vauldSlots, w); err != nil {
		return err
	}
	if err = f.backupMessages(vauldSlots, w); err != nil {
		return err
	}
	return nil
}

func (f *FileDB) backupBaseData(slots []uint32, w io.Writer) error {

	return f.db.View(func(t *bolt.Tx) error {
		for _, slot := range slots {
			encoder := lmproto.NewEncoder()

			b, err := f.getSlotBucket(slot, t)
			if err != nil {
				return err
			}
			encoder.WriteBytes(BackupSlotMagicNumber[:])
			encoder.WriteUint32(slot)
			encoder.WriteInt32(int32(b.Stats().KeyN))

			err = b.ForEach(func(k, v []byte) error {
				encoder.WriteBinary(k)
				encoder.WriteBinary(v)
				return nil
			})
			if err != nil {
				return err
			}
			_, err = w.Write(encoder.Bytes())
			if err != nil {
				return err
			}

		}
		return nil
	})
}

func (f *FileDB) backupMessages(slots []uint32, w io.Writer) error {

	segmentInfos, err := f.getSegmentInfo(slots)
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
				has, err := f.scanner.Scan(segmentPath, 0, func(data []byte) error {
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
	return nil
}

// RecoverSlotBackup 恢复备份
func (f *FileDB) RecoverSlotBackup(reader io.Reader) error {
	magicNumber := make([]byte, len(BackupMagicNumber))
	_, err := reader.Read(magicNumber)
	if err != nil {
		return err
	}
	if !bytes.Equal(magicNumber, BackupMagicNumber[:]) {
		return fmt.Errorf("BackupMagicNumber is error 期望:%x 实际:%x", BackupMagicNumber[:], magicNumber)
	}

	slots, err := lmproto.Binary(reader)
	if err != nil {
		f.Error("读取slots失败！", zap.Error(err))
		return err
	}
	slotBitM := util.NewSlotBitMapWithBits(slots)
	vaildSlots := slotBitM.GetVaildSlots()
	if len(vaildSlots) == 0 {
		return nil
	}

	err = f.recoverBaseData(reader, vaildSlots)
	if err != nil {
		return err
	}
	return f.recoverMessages(reader, vaildSlots)
}

func (f *FileDB) recoverBaseData(reader io.Reader, vaildSlots []uint32) error {

	for i := 0; i < len(vaildSlots); i++ {
		slot, keyN, err := f.readBaseDataHeader(reader)
		if err != nil {
			return err
		}
		err = f.db.Batch(func(t *bolt.Tx) error {
			for j := 0; j < int(keyN); j++ {
				key, err := lmproto.Binary(reader)
				if err != nil {
					return err
				}
				value, err := lmproto.Binary(reader)
				if err != nil {
					return err
				}
				b, err := f.getSlotBucket(slot, t)
				if err != nil {
					return err
				}
				err = b.Put(key, value)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *FileDB) recoverMessages(reader io.Reader, vaildSlots []uint32) error {
	if err := f.checkAndBackupSlotDir(vaildSlots); err != nil {
		f.Error("检查slot目录失败！", zap.Error(err))
		return err
	}
	for {
		err := f.recoverMessage(reader)
		if err != nil {
			if errors.Is(err, ErrorReadFinished) {
				return nil
			}
			return err
		}
	}
}

// 检查slot目录 如果有数据则备份原来数据（理论上不应该有数据，如果有的话则数据不知道那来的，管他三七二十一备份再说）
func (f *FileDB) checkAndBackupSlotDir(vaildSlots []uint32) error {
	slotRootDir := filepath.Join(f.dataDir, "slots")
	for _, s := range vaildSlots {
		slotDir := path.Join(slotRootDir, fmt.Sprintf("%d", s))
		_, err := os.Stat(slotDir)
		if err != nil && os.IsExist(err) {
			err = os.Rename(slotDir, fmt.Sprintf("%s%s%s", slotDir, "_bak", fmt.Sprintf("%d", time.Now().Unix())))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *FileDB) recoverMessage(reader io.Reader) error {
	magicNumber := make([]byte, len(MagicNumber))
	_, err := reader.Read(magicNumber)
	if err != nil {
		if err == io.EOF {
			return ErrorReadFinished
		}
		return nil
	}

	if !bytes.Equal(magicNumber, MagicNumber[:]) {
		err = fmt.Errorf("消息开始魔法数错误 期望:%x 实际:%x", MagicNumber[:], magicNumber)
		return err
	}
	logVersion := make([]byte, 1)
	_, err = reader.Read(logVersion)
	if err != nil {
		return nil
	}
	var dataLen uint32
	if err = binary.Read(reader, Encoding, &dataLen); err != nil {
		return err
	}
	var offset int64
	if err = binary.Read(reader, Encoding, &offset); err != nil {
		return err
	}
	var appliIndex uint64
	if err = binary.Read(reader, Encoding, &appliIndex); err != nil {
		return err
	}
	data := make([]byte, dataLen)
	if _, err = reader.Read(data); err != nil {
		return err
	}
	endMagicNumber := make([]byte, len(EndMagicNumber))
	if _, err = reader.Read(endMagicNumber); err != nil {
		return err
	}
	if !bytes.Equal(endMagicNumber, EndMagicNumber[:]) {
		err = fmt.Errorf("消息结尾魔法数错误 期望:%x 实际:%x", EndMagicNumber[:], endMagicNumber)
		return err
	}
	var message = &Message{}
	err = UnmarshalMessage(data, message)
	if err != nil {
		return err
	}
	message.AppliIndex = 0
	if strings.TrimSpace(message.QueueUID) != "" {
		_, err = f.AppendMessageOfUser(message)
	} else {
		_, err = f.AppendMessage(message)
	}

	return err

}

func (f *FileDB) readBaseDataHeader(reader io.Reader) (slot uint32, keyN int32, err error) {
	magicNumber := make([]byte, len(BackupSlotMagicNumber))
	_, err = reader.Read(magicNumber)
	if err != nil {
		return
	}
	if !bytes.Equal(magicNumber, BackupSlotMagicNumber[:]) {
		err = fmt.Errorf("BackupSlotMagicNumber is error 期望:%x 实际:%x", BackupSlotMagicNumber[:], magicNumber)
		return
	}
	slot, err = lmproto.Uint32(reader)
	if err != nil {
		return
	}
	keyN, err = lmproto.Int32(reader)
	return
}

// AddNodeInFlightData 添加节点inflight数据
func (f *FileDB) AddNodeInFlightData(data []*NodeInFlightDataModel) error {
	if len(data) <= 0 {
		return nil
	}
	return f.db.Update(func(t *bbolt.Tx) error {
		bucket := f.getRootBucket(t)
		return bucket.Put([]byte(f.nodeInFlightDataPrefix), []byte(util.ToJSON(data)))
	})
}

// GetNodeInFlightData 获取投递给节点的inflight数据
func (f *FileDB) GetNodeInFlightData() ([]*NodeInFlightDataModel, error) {
	var data = make([]*NodeInFlightDataModel, 0)
	err := f.db.Update(func(t *bbolt.Tx) error {
		bucket := f.getRootBucket(t)

		value := bucket.Get([]byte(f.nodeInFlightDataPrefix))
		if len(value) > 0 {
			err := util.ReadJSONByByte(value, &data)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return data, err
}

// ClearNodeInFlightData ClearNodeInFlightData
func (f *FileDB) ClearNodeInFlightData() error {
	return f.db.Update(func(t *bbolt.Tx) error {
		bucket := f.getRootBucket(t)
		return bucket.Delete([]byte(f.nodeInFlightDataPrefix))
	})
}
