package lim

import (
	"fmt"

	"github.com/lim-team/LiMaoIM/internal/db"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
)

// Conversation Conversation

// Storage Storage
type Storage interface {
	GetUserToken(uid string, deviceFlag lmproto.DeviceFlag) (string, lmproto.DeviceLevel, error)
	UpdateUserToken(uid string, deviceFlag lmproto.DeviceFlag, deviceLevel lmproto.DeviceLevel, token string) error
	// ---------- channel ----------
	GetChannel(channelID string, channelType uint8) (*ChannelInfo, error)
	ExistChannel(channelID string, channelType uint8) (bool, error)
	DeleteChannel(channelID string, channelType uint8) error
	AddOrUpdateChannel(channelInfo *ChannelInfo) error
	AddSubscribers(channelID string, channelType uint8, uids []string) error
	GetSubscribers(channelID string, channelType uint8) ([]string, error)
	RemoveAllSubscriber(channelID string, channelType uint8) error
	RemoveSubscribers(channelID string, channelType uint8, uids []string) error
	// ---------- denylist ----------
	AddDenylist(channelID string, channelType uint8, uids []string) error
	GetDenylist(channelID string, channelType uint8) ([]string, error)
	RemoveDenylist(channelID string, channelType uint8, uids []string) error
	RemoveAllDenylist(channelID string, channelType uint8) error
	// ---------- allowlist ----------
	GetAllowlist(channelID string, channelType uint8) ([]string, error)
	AddAllowlist(channelID string, channelType uint8, uids []string) error
	RemoveAllowlist(channelID string, channelType uint8, uids []string) error
	RemoveAllAllowlist(channelID string, channelType uint8) error
	// ---------- message ----------
	GetNextMessageSeq(channelID string, channelType uint8) (uint32, error)
	AppendMessage(m *db.Message) (n int, err error)
	GetMessages(channelID string, channelType uint8, offset uint32, limit uint64) ([]*db.Message, error)
	GetMessagesWithOptions(channelID string, channelType uint8, offset uint32, limit uint64, reverse bool, endMessageSeq uint32) ([]*db.Message, error)
	GetMessage(channelID string, channelType uint8, messageSeq uint32) (*db.Message, error)
	AppendMessageOfUser(uid string, m *db.Message) (int, error)
	AppendMessageOfNotifyQueue(m *db.Message) error
	GetMessagesOfNotifyQueue(count int) ([]*db.Message, error)
	RemoveMessagesOfNotifyQueue(messageIDs []int64) error
	// ---------- conversation ----------
	AddOrUpdateConversations(uid string, conversations []*db.Conversation) error
	GetConversations(uid string) ([]*db.Conversation, error)

	Close() error
}

// DefaultStorage DefaultStorage
type DefaultStorage struct {
	l  *LiMao
	db db.DB // Storage interface
}

// NewStorage NewStorage
func NewStorage(l *LiMao) Storage {
	return &DefaultStorage{
		l:  l,
		db: db.NewFileDB(l.opts.DataDir, l.opts.SegmentMaxBytes, l.opts.SlotCount),
	}
}

// GetUserToken GetUserToken
func (d *DefaultStorage) GetUserToken(uid string, deviceFlag lmproto.DeviceFlag) (string, lmproto.DeviceLevel, error) {

	return d.db.GetUserToken(uid, deviceFlag)
}

// UpdateUserToken UpdateUserToken
func (d *DefaultStorage) UpdateUserToken(uid string, deviceFlag lmproto.DeviceFlag, deviceLevel lmproto.DeviceLevel, token string) error {
	return d.db.UpdateUserToken(uid, deviceFlag, deviceLevel, token)
}

// GetChannel GetChannel
func (d *DefaultStorage) GetChannel(channelID string, channelType uint8) (*ChannelInfo, error) {
	channelMap, err := d.db.GetChannel(channelID, channelType)
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
func (d *DefaultStorage) AddOrUpdateChannel(channelInfo *ChannelInfo) error {
	return d.db.AddOrUpdateChannel(channelInfo.ChannelID, channelInfo.ChannelType, channelInfo.ToMap())
}

// AddSubscribers AddSubscribers
func (d *DefaultStorage) AddSubscribers(channelID string, channelType uint8, uids []string) error {
	return d.db.AddSubscribers(channelID, channelType, uids)
}

// RemoveAllSubscriber RemoveAllSubscriber
func (d *DefaultStorage) RemoveAllSubscriber(channelID string, channelType uint8) error {
	return d.db.RemoveAllSubscriber(channelID, channelType)
}

// RemoveSubscribers RemoveSubscribers
func (d *DefaultStorage) RemoveSubscribers(channelID string, channelType uint8, uids []string) error {
	return d.db.RemoveSubscribers(channelID, channelType, uids)
}

// GetSubscribers GetSubscribers
func (d *DefaultStorage) GetSubscribers(channelID string, channelType uint8) ([]string, error) {
	return d.db.GetSubscribers(channelID, channelType)
}

// DeleteChannel DeleteChannel
func (d *DefaultStorage) DeleteChannel(channelID string, channelType uint8) error {
	return d.db.DeleteChannel(channelID, channelType)
}

// ExistChannel ExistChannel
func (d *DefaultStorage) ExistChannel(channelID string, channelType uint8) (bool, error) {
	return d.db.ExistChannel(channelID, channelType)
}

// GetNextMessageSeq GetNextMessageSeq
func (d *DefaultStorage) GetNextMessageSeq(channelID string, channelType uint8) (uint32, error) {
	return d.db.GetNextMessageSeq(channelID, channelType)
}

// AppendMessage AppendMessage
func (d *DefaultStorage) AppendMessage(m *db.Message) (n int, err error) {
	return d.db.AppendMessage(m)
}

// AppendMessageOfUser AppendMessageOfUser
func (d *DefaultStorage) AppendMessageOfUser(uid string, m *db.Message) (int, error) {
	return d.db.AppendMessageOfUser(uid, m)
}

// AppendMessageOfNotifyQueue AppendMessageOfNotifyQueue
func (d *DefaultStorage) AppendMessageOfNotifyQueue(m *db.Message) error {
	return d.db.AppendMessageOfNotifyQueue(m)
}

// GetMessagesOfNotifyQueue GetMessagesOfNotifyQueue
func (d *DefaultStorage) GetMessagesOfNotifyQueue(count int) ([]*db.Message, error) {
	return d.db.GetMessagesOfNotifyQueue(count)
}

// RemoveMessagesOfNotifyQueue RemoveMessagesOfNotifyQueue
func (d *DefaultStorage) RemoveMessagesOfNotifyQueue(messageIDs []int64) error {
	return d.db.RemoveMessagesOfNotifyQueue(messageIDs)
}

// GetMessages GetMessages
func (d *DefaultStorage) GetMessages(channelID string, channelType uint8, offset uint32, limit uint64) ([]*db.Message, error) {
	return d.db.GetMessages(channelID, channelType, offset, limit)
}

// GetMessagesWithOptions GetMessagesWithOptions
func (d *DefaultStorage) GetMessagesWithOptions(channelID string, channelType uint8, offset uint32, limit uint64, reverse bool, endMessageSeq uint32) ([]*db.Message, error) {
	if reverse {
		offsetSeq := int64(offset) - int64(limit) - 1
		if offsetSeq < 0 {
			offsetSeq = 0
		}
		messages, err := d.db.GetMessages(channelID, channelType, uint32(offsetSeq), limit)
		if err != nil {
			return nil, err
		}
		newMessages := make([]*db.Message, 0, len(messages))
		if len(messages) > 0 {
			for _, message := range messages {
				if message.MessageSeq < offset {
					newMessages = append(newMessages, message)
				}
			}
		}
		return newMessages, nil
	} else {
		messages, err := d.db.GetMessages(channelID, channelType, offset+1, limit)
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
func (d *DefaultStorage) GetMessage(channelID string, channelType uint8, messageSeq uint32) (*db.Message, error) {
	return d.db.GetMessage(channelID, channelType, messageSeq)
}

// AddOrUpdateConversations AddOrUpdateConversations
func (d *DefaultStorage) AddOrUpdateConversations(uid string, conversations []*db.Conversation) error {
	fmt.Println("AddOrUpdateConversations---->", uid, conversations)
	return d.db.AddOrUpdateConversations(uid, conversations)
}

// GetConversations GetConversations
func (d *DefaultStorage) GetConversations(uid string) ([]*db.Conversation, error) {
	return d.db.GetConversations(uid)
}

// Close Close
func (d *DefaultStorage) Close() error {
	return d.db.Close()
}

// AddDenylist AddDenylist
func (d *DefaultStorage) AddDenylist(channelID string, channelType uint8, uids []string) error {
	return d.db.AddDenylist(channelID, channelType, uids)
}

// GetDenylist GetDenylist
func (d *DefaultStorage) GetDenylist(channelID string, channelType uint8) ([]string, error) {
	return d.db.GetDenylist(channelID, channelType)
}

// RemoveDenylist RemoveDenylist
func (d *DefaultStorage) RemoveDenylist(channelID string, channelType uint8, uids []string) error {
	return d.db.RemoveDenylist(channelID, channelType, uids)
}

// RemoveAllDenylist RemoveAllDenylist
func (d *DefaultStorage) RemoveAllDenylist(channelID string, channelType uint8) error {
	return d.db.RemoveAllDenylist(channelID, channelType)
}

// GetAllowlist 获取白名单
func (d *DefaultStorage) GetAllowlist(channelID string, channelType uint8) ([]string, error) {
	return d.db.GetAllowlist(channelID, channelType)
}

// AddAllowlist 添加白名单
func (d *DefaultStorage) AddAllowlist(channelID string, channelType uint8, uids []string) error {

	return d.db.AddAllowlist(channelID, channelType, uids)
}

// RemoveAllowlist 移除白名单
func (d *DefaultStorage) RemoveAllowlist(channelID string, channelType uint8, uids []string) error {
	return d.db.RemoveAllowlist(channelID, channelType, uids)
}

// RemoveAllAllowlist 移除指定频道的所有白名单
func (d *DefaultStorage) RemoveAllAllowlist(channelID string, channelType uint8) error {
	return d.RemoveAllAllowlist(channelID, channelType)
}
