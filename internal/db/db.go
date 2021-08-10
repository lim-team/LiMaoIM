package db

import (
	"fmt"
	"io"

	"github.com/lim-team/LiMaoIM/internal/lim/rpc"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
)

// NodeInFlightDataModel NodeInFlightDataModel
type NodeInFlightDataModel struct {
	No     string // 请求唯一编号
	NodeID int32  // 接受消息的节点ID
	Req    *rpc.ForwardRecvPacketReq
}

// Conversation Conversation
type Conversation struct {
	UID             string // User UID (user who belongs to the most recent session)
	ChannelID       string // Conversation channel
	ChannelType     uint8
	UnreadCount     int    // Number of unread messages
	Timestamp       int64  // Last session timestamp (10 digits)
	LastMsgSeq      uint32 // Sequence number of the last message
	LastClientMsgNo string // Last message client number
	LastMsgID       int64  // Last message ID
	Version         int64  // Data version
}

func (c *Conversation) String() string {
	return fmt.Sprintf("uid:%s channelID:%s channelType:%d unreadCount:%d timestamp: %d lastMsgSeq:%d lastClientMsgNo:%s lastMsgID:%d version:%d", c.UID, c.ChannelID, c.ChannelType, c.UnreadCount, c.Timestamp, c.LastMsgSeq, c.LastClientMsgNo, c.LastMsgID, c.Version)
}

// DB DB
type DB interface {
	Open() error
	Close() error
	// SaveMetaData  Application index of storage raft
	SaveMetaData(appliIndex uint64) error
	// GetMetaData Get Application index
	GetMetaData() (uint64, error)
	// GetUserToken 获取用户指定设备的token
	GetUserToken(uid string, deviceFlag lmproto.DeviceFlag) (token string, level lmproto.DeviceLevel, err error)
	// UpdateUserToken 更新用户指定设备的token
	UpdateUserToken(uid string, deviceFlag lmproto.DeviceFlag, deviceLevel lmproto.DeviceLevel, token string) error

	AddOrUpdateConversations(uid string, conversations []*Conversation) error
	GetConversations(uid string) ([]*Conversation, error)

	// 添加节点inflight数据
	AddNodeInFlightData(data []*NodeInFlightDataModel) error
	// 获取投递给节点的inflight数据
	GetNodeInFlightData() ([]*NodeInFlightDataModel, error)
	ClearNodeInFlightData() error

	// BackupSlots 备份slots
	BackupSlots(slots []byte, w io.Writer) error
	// RecoverSlotBackup 恢复备份
	RecoverSlotBackup(reader io.Reader) error

	IMessageDB
	IChannelDB
	IDenyAndAllowlistStore
	ISnapshot
}

// IMessageDB IMessageDB
type IMessageDB interface {
	GetUserNextMessageSeq(uid string) (uint32, error)
	// AppendMessage 追加消息到频道队列  n 为追加的实际字节数
	AppendMessage(m *Message) (n int, err error)

	// AppendMessageOfUser 追加消息到用户队列
	AppendMessageOfUser(m *Message) (int, error)
	// UpdateMessageOfUserCursorIfNeed 更新用户消息队列的游标，用户读到的位置
	UpdateMessageOfUserCursorIfNeed(uid string, offset uint32) error
	// GetMessageOfUserCursor(uid string) (uint32, error)
	// AppendMessageOfNotifyQueue 追加消息到通知队列
	AppendMessageOfNotifyQueue(m *Message) error
	GetMessagesOfNotifyQueue(count int) ([]*Message, error)
	// RemoveMessagesOfNotifyQueue 从通知队列里移除消息
	RemoveMessagesOfNotifyQueue(messageIDs []int64) error
	// GetMessages 获取消息
	GetMessages(channelID string, channelType uint8, offset uint32, limit uint64) ([]*Message, error)
	GetLastMessages(channelID string, channelType uint8, endOffset uint32, limit uint64) ([]*Message, error)
	GetMessagesOfUser(uid string, offset uint32, limit uint64) ([]*Message, error)
	GetMessage(channelID string, channelType uint8, messageSeq uint32) (*Message, error)
	DeleteMessages(channelID string, channelType uint8) error
}

// IChannelDB IChannelDB
type IChannelDB interface {
	// GetNextMessageSeq 获取频道的下一个序号
	GetNextMessageSeq(channelID string, channelType uint8) (uint32, error)
	// AddOrUpdateChannel 添加或者更新频道
	AddOrUpdateChannel(channelID string, channelType uint8, data map[string]interface{}) error
	// GetChannel 获取频道数据
	GetChannel(channelID string, channelType uint8) (map[string]interface{}, error)
	// DeleteChannel 删除频道
	DeleteChannel(channelID string, channelType uint8) error
	DeleteChannelAndClearMessages(channelID string, channelType uint8) error
	// ExistChannel 是否存在指定的频道
	ExistChannel(channelID string, channelType uint8) (bool, error)
	// AddSubscribers 添加订阅者
	AddSubscribers(channelID string, channelType uint8, uids []string) error
	// RemoveSubscribers 移除指定频道内指定uid的订阅者
	RemoveSubscribers(channelID string, channelType uint8, uids []string) error
	// GetSubscribers 获取订阅者列表
	GetSubscribers(channelID string, channelType uint8) ([]string, error)
	RemoveAllSubscriber(channelID string, channelType uint8) error
}

// IDenyAndAllowlistStore IDenyAndAllowlistStore
type IDenyAndAllowlistStore interface {
	// AddDenylist 添加频道黑名单
	AddDenylist(channelID string, channelType uint8, uids []string) error
	// GetDenylist 获取频道黑名单列表
	GetDenylist(channelID string, channelType uint8) ([]string, error)
	// RemoveDenylist 移除频道内指定用户的黑名单
	RemoveDenylist(channelID string, channelType uint8, uids []string) error
	// RemoveAllDenylist 移除指定频道的所有黑名单
	RemoveAllDenylist(channelID string, channelType uint8) error
	// GetAllowlist 获取白名单
	GetAllowlist(channelID string, channelType uint8) ([]string, error)
	// AddAllowlist 添加白名单
	AddAllowlist(channelID string, channelType uint8, uids []string) error
	// RemoveAllowlist 移除白名单
	RemoveAllowlist(channelID string, channelType uint8, uids []string) error
	// RemoveAllAllowlist 移除指定频道的所有白名单
	RemoveAllAllowlist(channelID string, channelType uint8) error
}

// ISnapshot ISnapshot
type ISnapshot interface {
	// PrepareSnapshot PrepareSnapshot
	PrepareSnapshot() (*Snapshot, error)
	// SaveSnapshot SaveSnapshot
	SaveSnapshot(snapshot *Snapshot, w io.Writer) error
}
