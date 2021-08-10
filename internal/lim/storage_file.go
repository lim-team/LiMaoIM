package lim

import (
	"io"

	"github.com/lim-team/LiMaoIM/internal/db"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
)

// FileStorage FileStorage
type FileStorage struct {
	l  *LiMao
	db db.DB // Storage interface

}

// NewFileStorage NewFileStorage
func NewFileStorage(l *LiMao) *FileStorage {
	return &FileStorage{
		l:  l,
		db: db.NewFileDB(l.opts.DataDir, l.opts.SegmentMaxBytes, l.opts.SlotCount),
	}
}

// GetFileStorage GetFileStorage
func (f *FileStorage) GetFileStorage() *FileStorage {
	return f
}

// SaveMetaData SaveMetaData
func (f *FileStorage) SaveMetaData(appliIndex uint64) error {

	return f.db.SaveMetaData(appliIndex)
}

// GetMetaData GetMetaData
func (f *FileStorage) GetMetaData() (uint64, error) {
	return f.db.GetMetaData()
}

func (f *FileStorage) isCluster() bool {
	return f.l.opts.IsCluster
}

// GetUserToken GetUserToken
func (f *FileStorage) GetUserToken(uid string, deviceFlag lmproto.DeviceFlag) (string, lmproto.DeviceLevel, error) {

	return f.db.GetUserToken(uid, deviceFlag)
}

// UpdateUserToken UpdateUserToken
func (f *FileStorage) UpdateUserToken(uid string, deviceFlag lmproto.DeviceFlag, deviceLevel lmproto.DeviceLevel, token string) error {

	return f.db.UpdateUserToken(uid, deviceFlag, deviceLevel, token)
}

// GetChannel GetChannel
func (f *FileStorage) GetChannel(channelID string, channelType uint8) (*ChannelInfo, error) {
	channelMap, err := f.db.GetChannel(channelID, channelType)
	if err != nil {
		return nil, err
	}
	channelInfo := &ChannelInfo{}
	channelInfo.ChannelID = channelID
	channelInfo.ChannelType = channelType
	channelInfo.from(channelMap)
	return channelInfo, nil
}

// AddOrUpdateChannel AddOrUpdateChannel
func (f *FileStorage) AddOrUpdateChannel(channelInfo *ChannelInfo) error {
	return f.db.AddOrUpdateChannel(channelInfo.ChannelID, channelInfo.ChannelType, channelInfo.ToMap())
}

// AddSubscribers AddSubscribers
func (f *FileStorage) AddSubscribers(channelID string, channelType uint8, uids []string) error {
	return f.db.AddSubscribers(channelID, channelType, uids)
}

// RemoveAllSubscriber RemoveAllSubscriber
func (f *FileStorage) RemoveAllSubscriber(channelID string, channelType uint8) error {
	return f.db.RemoveAllSubscriber(channelID, channelType)
}

// RemoveSubscribers RemoveSubscribers
func (f *FileStorage) RemoveSubscribers(channelID string, channelType uint8, uids []string) error {
	return f.db.RemoveSubscribers(channelID, channelType, uids)
}

// GetSubscribers GetSubscribers
func (f *FileStorage) GetSubscribers(channelID string, channelType uint8) ([]string, error) {
	return f.db.GetSubscribers(channelID, channelType)
}

// DeleteChannel DeleteChannel
func (f *FileStorage) DeleteChannel(channelID string, channelType uint8) error {
	return f.db.DeleteChannel(channelID, channelType)
}

// DeleteChannelAndClearMessages DeleteChannelAndClearMessages
func (f *FileStorage) DeleteChannelAndClearMessages(channelID string, channelType uint8) error {

	return f.db.DeleteChannelAndClearMessages(channelID, channelType)
}

// ExistChannel ExistChannel
func (f *FileStorage) ExistChannel(channelID string, channelType uint8) (bool, error) {
	return f.db.ExistChannel(channelID, channelType)
}

// GetNextMessageSeq GetNextMessageSeq
func (f *FileStorage) GetNextMessageSeq(channelID string, channelType uint8) (uint32, error) {
	return f.db.GetNextMessageSeq(channelID, channelType)
}

// GetUserNextMessageSeq GetUserNextMessageSeq
func (f *FileStorage) GetUserNextMessageSeq(uid string) (uint32, error) {
	return f.db.GetUserNextMessageSeq(uid)
}

// AppendMessage AppendMessage
func (f *FileStorage) AppendMessage(m *db.Message) error {
	_, err := f.db.AppendMessage(m)
	return err
}

// AppendMessageOfUser AppendMessageOfUser
func (f *FileStorage) AppendMessageOfUser(m *db.Message) error {
	_, err := f.db.AppendMessageOfUser(m)
	return err
}

// AppendMessageOfNotifyQueue AppendMessageOfNotifyQueue
func (f *FileStorage) AppendMessageOfNotifyQueue(m *db.Message) error {
	return f.db.AppendMessageOfNotifyQueue(m)
}

// GetMessagesOfNotifyQueue GetMessagesOfNotifyQueue
func (f *FileStorage) GetMessagesOfNotifyQueue(count int) ([]*db.Message, error) {
	return f.db.GetMessagesOfNotifyQueue(count)
}

// RemoveMessagesOfNotifyQueue RemoveMessagesOfNotifyQueue
func (f *FileStorage) RemoveMessagesOfNotifyQueue(messageIDs []int64) error {
	return f.db.RemoveMessagesOfNotifyQueue(messageIDs)
}

// GetMessages GetMessages
func (f *FileStorage) GetMessages(channelID string, channelType uint8, offset uint32, limit uint64) ([]*db.Message, error) {
	return f.db.GetMessages(channelID, channelType, offset, limit)
}

// GetMessagesOfUser GetMessagesOfUser
func (f *FileStorage) GetMessagesOfUser(uid string, offset uint32, limit uint64) ([]*db.Message, error) {
	return f.db.GetMessagesOfUser(uid, offset, limit)
}

// UpdateMessageOfUserCursorIfNeed UpdateMessageOfUserCursorIfNeed
func (f *FileStorage) UpdateMessageOfUserCursorIfNeed(uid string, offset uint32) error {
	return f.db.UpdateMessageOfUserCursorIfNeed(uid, offset)
}

// GetMessagesWithOptions GetMessagesWithOptions
func (f *FileStorage) GetMessagesWithOptions(channelID string, channelType uint8, offsetMessageSeq uint32, limit uint64, reverse bool, endMessageSeq uint32) ([]*db.Message, error) {
	if reverse { // 上拉 messageSeq从小到大
		offsetSeq := int64(offsetMessageSeq) + 1

		if offsetSeq < 0 {
			offsetSeq = 0
		}
		actLimit := limit
		if endMessageSeq != 0 && endMessageSeq > offsetMessageSeq {
			actLimit = uint64(endMessageSeq - offsetMessageSeq)
			if actLimit > limit {
				actLimit = limit
			}
		}
		messages, err := f.db.GetMessages(channelID, channelType, uint32(offsetSeq), actLimit)
		if err != nil {
			return nil, err
		}
		newMessages := make([]*db.Message, 0, len(messages))
		if len(messages) > 0 {
			for _, message := range messages {
				if endMessageSeq == 0 || message.MessageSeq < endMessageSeq {
					newMessages = append(newMessages, message)
				}
			}
		}
		return newMessages, nil
	} else { // 下拉 messageSeq从大到小
		var messages []*db.Message
		var err error
		if offsetMessageSeq == 0 {
			offset := int64(endMessageSeq) - int64(limit)
			if offset < 0 {
				offset = 0
			}
			messages, err = f.db.GetMessages(channelID, channelType, uint32(offset), limit)
		} else {
			offsetSeq := offsetMessageSeq + 1
			messages, err = f.db.GetMessages(channelID, channelType, offsetSeq, limit)
		}
		if err != nil {
			return nil, err
		}
		if endMessageSeq == 0 {
			return messages, nil
		}
		newMessages := make([]*db.Message, 0, len(messages))
		for _, message := range messages {
			if message.MessageSeq < endMessageSeq {
				newMessages = append(newMessages, message)
			}
		}
		return newMessages, nil
	}
}

// GetMessage GetMessage
func (f *FileStorage) GetMessage(channelID string, channelType uint8, messageSeq uint32) (*db.Message, error) {
	return f.db.GetMessage(channelID, channelType, messageSeq)
}

// GetLastMessages GetLastMessages
func (f *FileStorage) GetLastMessages(channelID string, channelType uint8, endOffset uint32, limit uint64) ([]*db.Message, error) {
	return f.db.GetLastMessages(channelID, channelType, endOffset, limit)
}

// AddOrUpdateConversations AddOrUpdateConversations
func (f *FileStorage) AddOrUpdateConversations(uid string, conversations []*db.Conversation) error {
	return f.db.AddOrUpdateConversations(uid, conversations)
}

// GetConversations GetConversations
func (f *FileStorage) GetConversations(uid string) ([]*db.Conversation, error) {
	return f.db.GetConversations(uid)
}

// Close Close
func (f *FileStorage) Close() error {
	return f.db.Close()
}

// AddDenylist AddDenylist
func (f *FileStorage) AddDenylist(channelID string, channelType uint8, uids []string) error {
	return f.db.AddDenylist(channelID, channelType, uids)
}

// GetDenylist GetDenylist
func (f *FileStorage) GetDenylist(channelID string, channelType uint8) ([]string, error) {
	return f.db.GetDenylist(channelID, channelType)
}

// RemoveDenylist RemoveDenylist
func (f *FileStorage) RemoveDenylist(channelID string, channelType uint8, uids []string) error {
	return f.db.RemoveDenylist(channelID, channelType, uids)
}

// RemoveAllDenylist RemoveAllDenylist
func (f *FileStorage) RemoveAllDenylist(channelID string, channelType uint8) error {
	return f.db.RemoveAllDenylist(channelID, channelType)
}

// GetAllowlist 获取白名单
func (f *FileStorage) GetAllowlist(channelID string, channelType uint8) ([]string, error) {
	return f.db.GetAllowlist(channelID, channelType)
}

// AddAllowlist 添加白名单
func (f *FileStorage) AddAllowlist(channelID string, channelType uint8, uids []string) error {

	return f.db.AddAllowlist(channelID, channelType, uids)
}

// RemoveAllowlist 移除白名单
func (f *FileStorage) RemoveAllowlist(channelID string, channelType uint8, uids []string) error {
	return f.db.RemoveAllowlist(channelID, channelType, uids)
}

// RemoveAllAllowlist 移除指定频道的所有白名单
func (f *FileStorage) RemoveAllAllowlist(channelID string, channelType uint8) error {
	return f.db.RemoveAllAllowlist(channelID, channelType)
}

// AddNodeInFlightData 添加节点inflight数据
func (f *FileStorage) AddNodeInFlightData(data []*db.NodeInFlightDataModel) error {
	return f.db.AddNodeInFlightData(data)
}

// GetNodeInFlightData GetNodeInFlightData
func (f *FileStorage) GetNodeInFlightData() ([]*db.NodeInFlightDataModel, error) {
	return f.db.GetNodeInFlightData()
}

// ClearNodeInFlightData 清除inflight数据
func (f *FileStorage) ClearNodeInFlightData() error {

	return f.db.ClearNodeInFlightData()
}

// BackupSlots BackupSlots
func (f *FileStorage) BackupSlots(slots []byte, w io.Writer) error {
	return f.db.BackupSlots(slots, w)
}

// RecoverSlotBackup 恢复备份
func (f *FileStorage) RecoverSlotBackup(reader io.Reader) error {
	return f.db.RecoverSlotBackup(reader)
}
