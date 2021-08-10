package lim

import (
	"context"
	"time"

	"github.com/lim-team/LiMaoIM/internal/db"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
)

// ClusterStorage ClusterStorage
type ClusterStorage struct {
	l       *LiMao
	timeout time.Duration
}

// NewClusterStorage NewClusterStorage
func NewClusterStorage(l *LiMao) StorageWriter {

	c := &ClusterStorage{
		l:       l,
		timeout: time.Second * 50,
	}
	c.l.DoCommand = func(cmd *CMD) error {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), c.timeout)
		defer cancel()
		return l.clusterManager.SyncPropose(timeoutCtx, cmd.Encode())
	}
	return c
}

// UpdateUserToken UpdateUserToken
func (c *ClusterStorage) UpdateUserToken(uid string, deviceFlag lmproto.DeviceFlag, deviceLevel lmproto.DeviceLevel, token string) error {
	return c.l.DoCommand(NewCMD(CMDUpdateUserToken).EncodeUserToken(uid, deviceFlag, deviceLevel, token))
}

// AddOrUpdateChannel AddOrUpdateChannel
func (c *ClusterStorage) AddOrUpdateChannel(channelInfo *ChannelInfo) error {
	return c.l.DoCommand(NewCMD(CMDAddOrUpdateChannel).EncodeAddOrUpdateChannel(channelInfo))
}

// AddSubscribers AddSubscribers
func (c *ClusterStorage) AddSubscribers(channelID string, channelType uint8, uids []string) error {
	return c.l.DoCommand(NewCMD(CMDAddSubscribers).EncodeCMDAddSubscribers(channelID, channelType, uids))
}

// RemoveAllSubscriber RemoveAllSubscriber
func (c *ClusterStorage) RemoveAllSubscriber(channelID string, channelType uint8) error {
	return c.l.DoCommand(NewCMD(CMDRemoveAllSubscriber).EncodeRemoveAllSubscriber(channelID, channelType))
}

// RemoveSubscribers RemoveSubscribers
func (c *ClusterStorage) RemoveSubscribers(channelID string, channelType uint8, uids []string) error {
	return c.l.DoCommand(NewCMD(CMDRemoveSubscribers).EncodeCMDRemoveSubscribers(channelID, channelType, uids))
}

// DeleteChannel DeleteChannel
func (c *ClusterStorage) DeleteChannel(channelID string, channelType uint8) error {
	return c.l.DoCommand(NewCMD(CMDDeleteChannel).EncodeCMDDeleteChannel(channelID, channelType))
}

// DeleteChannelAndClearMessages DeleteChannelAndClearMessages
func (c *ClusterStorage) DeleteChannelAndClearMessages(channelID string, channelType uint8) error {

	return c.l.DoCommand(NewCMD(CMDDeleteChannelAndClearMessages).EncodeCMDDeleteChannelAndClearMessages(channelID, channelType))
}

// AppendMessage AppendMessage
func (c *ClusterStorage) AppendMessage(m *db.Message) error {
	return c.l.DoCommand(NewCMD(CMDAppendMessage).EncodeAppendMessage(m))
}

// AppendMessageOfUser AppendMessageOfUser
func (c *ClusterStorage) AppendMessageOfUser(m *db.Message) error {

	return c.l.DoCommand(NewCMD(CMDAppendMessageOfUser).EncodeCMDAppendMessageOfUser(m))
}

// UpdateMessageOfUserCursorIfNeed UpdateMessageOfUserCursorIfNeed
func (c *ClusterStorage) UpdateMessageOfUserCursorIfNeed(uid string, offset uint32) error {
	return c.l.DoCommand(NewCMD(CMDUpdateMessageOfUserCursorIfNeed).EncodeCMDUpdateMessageOfUserCursorIfNeed(uid, offset))
}

// AppendMessageOfNotifyQueue AppendMessageOfNotifyQueue
func (c *ClusterStorage) AppendMessageOfNotifyQueue(m *db.Message) error {
	return c.l.DoCommand(NewCMD(CMDAppendMessageOfNotifyQueue).EncodeCMDAppendMessageOfNotifyQueue(m))
}

// RemoveMessagesOfNotifyQueue RemoveMessagesOfNotifyQueue
func (c *ClusterStorage) RemoveMessagesOfNotifyQueue(messageIDs []int64) error {
	return c.l.DoCommand(NewCMD(CMDRemoveMessagesOfNotifyQueue).EncodeCMDRemoveMessagesOfNotifyQueue(messageIDs))
}

// AddOrUpdateConversations AddOrUpdateConversations
func (c *ClusterStorage) AddOrUpdateConversations(uid string, conversations []*db.Conversation) error {

	return c.l.DoCommand(NewCMD(CMDAddOrUpdateConversations).EncodeCMDAddOrUpdateConversations(uid, conversations))
}

// AddDenylist AddDenylist
func (c *ClusterStorage) AddDenylist(channelID string, channelType uint8, uids []string) error {
	return c.l.DoCommand(NewCMD(CMDAddDenylist).EncodeCMDAddDenylist(channelID, channelType, uids))
}

// RemoveDenylist RemoveDenylist
func (c *ClusterStorage) RemoveDenylist(channelID string, channelType uint8, uids []string) error {
	return c.l.DoCommand(NewCMD(CMDRemoveDenylist).EncodeCMDRemoveDenylist(channelID, channelType, uids))
}

// RemoveAllDenylist RemoveAllDenylist
func (c *ClusterStorage) RemoveAllDenylist(channelID string, channelType uint8) error {
	return c.l.DoCommand(NewCMD(CMDRemoveAllDenylist).EncodeCMDRemoveAllDenylist(channelID, channelType))
}

// AddAllowlist 添加白名单
func (c *ClusterStorage) AddAllowlist(channelID string, channelType uint8, uids []string) error {

	return c.l.DoCommand(NewCMD(CMDAddAllowlist).EncodeCMDAddAllowlist(channelID, channelType, uids))
}

// RemoveAllowlist 移除白名单
func (c *ClusterStorage) RemoveAllowlist(channelID string, channelType uint8, uids []string) error {
	return c.l.DoCommand(NewCMD(CMDRemoveAllowlist).EncodeCMDRemoveAllowlist(channelID, channelType, uids))
}

// RemoveAllAllowlist 移除指定频道的所有白名单
func (c *ClusterStorage) RemoveAllAllowlist(channelID string, channelType uint8) error {
	return c.l.DoCommand(NewCMD(CMDRemoveAllAllowlist).EncodeCMDRemoveAllAllowlist(channelID, channelType))
}

// AddNodeInFlightData 添加节点inflight数据
func (c *ClusterStorage) AddNodeInFlightData(data []*db.NodeInFlightDataModel) error {
	return c.l.DoCommand(NewCMD(CMDAddNodeInFlightData).EncodeCMDAddNodeInFlightData(data))
}

// ClearNodeInFlightData 清除inflight数据
func (c *ClusterStorage) ClearNodeInFlightData() error {
	return c.l.DoCommand(NewCMD(CMDClearNodeInFlightData).EncodeCMDClearNodeInFlightData())
}
