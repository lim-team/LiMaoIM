package lim

import (
	"context"
	"io"
	"time"

	"github.com/lim-team/LiMaoIM/internal/db"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
)

// Conversation Conversation

// Storage Storage
type Storage interface {
	StorageReader
	StorageWriter

	GetFileStorage() *FileStorage
}

// StorageReader StorageReader
type StorageReader interface {
	SaveMetaData(appliIndex uint64) error
	GetMetaData() (uint64, error)
	GetUserToken(uid string, deviceFlag lmproto.DeviceFlag) (string, lmproto.DeviceLevel, error)

	// ---------- channel ----------
	GetChannel(channelID string, channelType uint8) (*ChannelInfo, error)
	ExistChannel(channelID string, channelType uint8) (bool, error)

	GetSubscribers(channelID string, channelType uint8) ([]string, error)

	// ---------- denylist ----------
	GetDenylist(channelID string, channelType uint8) ([]string, error)

	// ---------- allowlist ----------
	GetAllowlist(channelID string, channelType uint8) ([]string, error)

	// ---------- message ----------
	GetNextMessageSeq(channelID string, channelType uint8) (uint32, error)
	GetUserNextMessageSeq(uid string) (uint32, error)
	GetMessages(channelID string, channelType uint8, offset uint32, limit uint64) ([]*db.Message, error)
	GetLastMessages(channelID string, channelType uint8, endOffset uint32, limit uint64) ([]*db.Message, error)
	GetMessagesWithOptions(channelID string, channelType uint8, offset uint32, limit uint64, reverse bool, endMessageSeq uint32) ([]*db.Message, error)
	GetMessagesOfUser(uid string, offset uint32, limit uint64) ([]*db.Message, error)
	GetMessage(channelID string, channelType uint8, messageSeq uint32) (*db.Message, error)
	GetMessagesOfNotifyQueue(count int) ([]*db.Message, error)

	// ---------- conversation ----------
	GetConversations(uid string) ([]*db.Conversation, error)

	// 获取投递给节点的inflight数据
	GetNodeInFlightData() ([]*db.NodeInFlightDataModel, error)

	BackupSlots(slots []byte, w io.Writer) error
	// RecoverSlotBackup 恢复备份
	RecoverSlotBackup(reader io.Reader) error

	Close() error
}

// StorageWriter StorageWriter
type StorageWriter interface {
	UpdateUserToken(uid string, deviceFlag lmproto.DeviceFlag, deviceLevel lmproto.DeviceLevel, token string) error

	// ---------- channel ----------
	DeleteChannel(channelID string, channelType uint8) error                 // 删除频道 但不清除消息
	DeleteChannelAndClearMessages(channelID string, channelType uint8) error // 删除频道并清除消息
	AddOrUpdateChannel(channelInfo *ChannelInfo) error
	AddSubscribers(channelID string, channelType uint8, uids []string) error
	RemoveAllSubscriber(channelID string, channelType uint8) error
	RemoveSubscribers(channelID string, channelType uint8, uids []string) error

	// ---------- denylist ----------
	AddDenylist(channelID string, channelType uint8, uids []string) error
	RemoveDenylist(channelID string, channelType uint8, uids []string) error
	RemoveAllDenylist(channelID string, channelType uint8) error

	// ---------- allowlist ----------
	AddAllowlist(channelID string, channelType uint8, uids []string) error
	RemoveAllowlist(channelID string, channelType uint8, uids []string) error
	RemoveAllAllowlist(channelID string, channelType uint8) error

	// ---------- message ----------
	AppendMessage(m *db.Message) error
	AppendMessageOfUser(m *db.Message) error
	AppendMessageOfNotifyQueue(m *db.Message) error
	RemoveMessagesOfNotifyQueue(messageIDs []int64) error
	UpdateMessageOfUserCursorIfNeed(uid string, offset uint32) error

	// ---------- conversation ----------
	AddOrUpdateConversations(uid string, conversations []*db.Conversation) error
	// 添加节点inflight数据
	AddNodeInFlightData(data []*db.NodeInFlightDataModel) error
	// 清除inflight数据
	ClearNodeInFlightData() error
}

// DefaultStorage DefaultStorage
type DefaultStorage struct {
	l *LiMao
	StorageReader
	StorageWriter
	ctx         context.Context
	timeout     time.Duration
	fileStorage *FileStorage
}

// NewStorage NewStorage
func NewStorage(l *LiMao) Storage {
	d := &DefaultStorage{
		l:       l,
		ctx:     context.Background(),
		timeout: time.Second * 50,
	}
	d.fileStorage = NewFileStorage(l)
	d.StorageReader = d.fileStorage
	d.StorageWriter = d.fileStorage
	if l.opts.IsCluster {
		d.StorageWriter = NewClusterStorage(l)
	}
	return d

}

// GetFileStorage GetFileStorage
func (d *DefaultStorage) GetFileStorage() *FileStorage {

	return d.fileStorage
}

// // SaveMetaData SaveMetaData
// func (d *DefaultStorage) SaveMetaData(appliIndex uint64) error {

// 	return d.store.SaveMetaData(appliIndex)
// }

// // GetMetaData GetMetaData
// func (d *DefaultStorage) GetMetaData() (uint64, error) {
// 	return d.store.GetMetaData()
// }

// // GetUserToken GetUserToken
// func (d *DefaultStorage) GetUserToken(uid string, deviceFlag lmproto.DeviceFlag) (string, lmproto.DeviceLevel, error) {

// 	return d.store.GetUserToken(uid, deviceFlag)
// }

// // UpdateUserToken UpdateUserToken
// func (d *DefaultStorage) UpdateUserToken(uid string, deviceFlag lmproto.DeviceFlag, deviceLevel lmproto.DeviceLevel, token string) error {
// 	return d.store.UpdateUserToken(uid, deviceFlag, deviceLevel, token)
// }

// // GetChannel GetChannel
// func (d *DefaultStorage) GetChannel(channelID string, channelType uint8) (*ChannelInfo, error) {
// 	channelInfo, err := d.store.GetChannel(channelID, channelType)
// 	return channelInfo, err
// }

// // AddOrUpdateChannel AddOrUpdateChannel
// func (d *DefaultStorage) AddOrUpdateChannel(channelInfo *ChannelInfo) error {

// 	return d.store.AddOrUpdateChannel(channelInfo)
// }

// // AddSubscribers AddSubscribers
// func (d *DefaultStorage) AddSubscribers(channelID string, channelType uint8, uids []string) error {
// 	return d.store.AddSubscribers(channelID, channelType, uids)
// }

// // RemoveAllSubscriber RemoveAllSubscriber
// func (d *DefaultStorage) RemoveAllSubscriber(channelID string, channelType uint8) error {
// 	return d.store.RemoveAllSubscriber(channelID, channelType)
// }

// // RemoveSubscribers RemoveSubscribers
// func (d *DefaultStorage) RemoveSubscribers(channelID string, channelType uint8, uids []string) error {
// 	return d.store.RemoveSubscribers(channelID, channelType, uids)
// }

// // GetSubscribers GetSubscribers
// func (d *DefaultStorage) GetSubscribers(channelID string, channelType uint8) ([]string, error) {
// 	return d.store.GetSubscribers(channelID, channelType)
// }

// // DeleteChannel DeleteChannel
// func (d *DefaultStorage) DeleteChannel(channelID string, channelType uint8) error {
// 	return d.store.DeleteChannel(channelID, channelType)
// }

// // DeleteChannelAndClearMessages DeleteChannelAndClearMessages
// func (d *DefaultStorage) DeleteChannelAndClearMessages(channelID string, channelType uint8) error {

// 	return d.store.DeleteChannelAndClearMessages(channelID, channelType)
// }

// // ExistChannel ExistChannel
// func (d *DefaultStorage) ExistChannel(channelID string, channelType uint8) (bool, error) {
// 	return d.store.ExistChannel(channelID, channelType)
// }

// // GetNextMessageSeq GetNextMessageSeq
// func (d *DefaultStorage) GetNextMessageSeq(channelID string, channelType uint8) (uint32, error) {
// 	return d.store.GetNextMessageSeq(channelID, channelType)
// }

// // GetUserNextMessageSeq GetUserNextMessageSeq
// func (d *DefaultStorage) GetUserNextMessageSeq(uid string) (uint32, error) {
// 	return d.store.GetUserNextMessageSeq(uid)
// }

// // AppendMessage AppendMessage
// func (d *DefaultStorage) AppendMessage(m *db.Message) (n int, err error) {
// 	return d.store.AppendMessage(m)
// }

// // AppendMessageOfUser AppendMessageOfUser
// func (d *DefaultStorage) AppendMessageOfUser(uid string, m *db.Message) (int, error) {
// 	return d.store.AppendMessageOfUser(uid, m)
// }

// // AppendMessageOfNotifyQueue AppendMessageOfNotifyQueue
// func (d *DefaultStorage) AppendMessageOfNotifyQueue(m *db.Message) error {
// 	return d.store.AppendMessageOfNotifyQueue(m)
// }

// // GetMessagesOfNotifyQueue GetMessagesOfNotifyQueue
// func (d *DefaultStorage) GetMessagesOfNotifyQueue(count int) ([]*db.Message, error) {
// 	return d.store.GetMessagesOfNotifyQueue(count)
// }

// // RemoveMessagesOfNotifyQueue RemoveMessagesOfNotifyQueue
// func (d *DefaultStorage) RemoveMessagesOfNotifyQueue(messageIDs []int64) error {
// 	return d.store.RemoveMessagesOfNotifyQueue(messageIDs)
// }

// // GetMessages GetMessages
// func (d *DefaultStorage) GetMessages(channelID string, channelType uint8, offset uint32, limit uint64) ([]*db.Message, error) {
// 	return d.store.GetMessages(channelID, channelType, offset, limit)
// }

// // GetMessagesWithOptions GetMessagesWithOptions
// func (d *DefaultStorage) GetMessagesWithOptions(channelID string, channelType uint8, offset uint32, limit uint64, reverse bool, endMessageSeq uint32) ([]*db.Message, error) {
// 	return d.store.GetMessagesWithOptions(channelID, channelType, offset, limit, reverse, endMessageSeq)
// }

// // GetMessage GetMessage
// func (d *DefaultStorage) GetMessage(channelID string, channelType uint8, messageSeq uint32) (*db.Message, error) {
// 	return d.store.GetMessage(channelID, channelType, messageSeq)
// }

// // AddOrUpdateConversations AddOrUpdateConversations
// func (d *DefaultStorage) AddOrUpdateConversations(uid string, conversations []*db.Conversation) error {
// 	return d.store.AddOrUpdateConversations(uid, conversations)
// }

// // GetConversations GetConversations
// func (d *DefaultStorage) GetConversations(uid string) ([]*db.Conversation, error) {
// 	return d.store.GetConversations(uid)
// }

// // Close Close
// func (d *DefaultStorage) Close() error {
// 	return d.store.Close()
// }

// // AddDenylist AddDenylist
// func (d *DefaultStorage) AddDenylist(channelID string, channelType uint8, uids []string) error {
// 	return d.store.AddDenylist(channelID, channelType, uids)
// }

// // GetDenylist GetDenylist
// func (d *DefaultStorage) GetDenylist(channelID string, channelType uint8) ([]string, error) {
// 	return d.store.GetDenylist(channelID, channelType)
// }

// // RemoveDenylist RemoveDenylist
// func (d *DefaultStorage) RemoveDenylist(channelID string, channelType uint8, uids []string) error {
// 	return d.store.RemoveDenylist(channelID, channelType, uids)
// }

// // RemoveAllDenylist RemoveAllDenylist
// func (d *DefaultStorage) RemoveAllDenylist(channelID string, channelType uint8) error {
// 	return d.store.RemoveAllDenylist(channelID, channelType)
// }

// // GetAllowlist 获取白名单
// func (d *DefaultStorage) GetAllowlist(channelID string, channelType uint8) ([]string, error) {
// 	return d.store.GetAllowlist(channelID, channelType)
// }

// // AddAllowlist 添加白名单
// func (d *DefaultStorage) AddAllowlist(channelID string, channelType uint8, uids []string) error {

// 	return d.store.AddAllowlist(channelID, channelType, uids)
// }

// // RemoveAllowlist 移除白名单
// func (d *DefaultStorage) RemoveAllowlist(channelID string, channelType uint8, uids []string) error {
// 	return d.store.RemoveAllowlist(channelID, channelType, uids)
// }

// // RemoveAllAllowlist 移除指定频道的所有白名单
// func (d *DefaultStorage) RemoveAllAllowlist(channelID string, channelType uint8) error {
// 	return d.store.RemoveAllAllowlist(channelID, channelType)
// }

// // AddNodeInFlightData 添加节点inflight数据
// func (d *DefaultStorage) AddNodeInFlightData(data []*db.NodeInFlightDataModel) error {
// 	return d.store.AddNodeInFlightData(data)
// }

// // GetNodeInFlightData GetNodeInFlightData
// func (d *DefaultStorage) GetNodeInFlightData() ([]*db.NodeInFlightDataModel, error) {
// 	return d.store.GetNodeInFlightData()
// }
