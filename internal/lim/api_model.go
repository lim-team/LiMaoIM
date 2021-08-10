package lim

import (
	"errors"
	"strconv"
	"strings"

	"github.com/lim-team/LiMaoIM/internal/db"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
)

type conversationResp struct {
	ChannelID   string       `json:"channel_id"`   // 频道ID
	ChannelType uint8        `json:"channel_type"` // 频道类型
	Unread      int          `json:"unread"`       // 未读数
	Timestamp   int64        `json:"timestamp"`
	LastMessage *MessageResp `json:"last_message"` // 最后一条消息
}

// MessageRespSlice MessageRespSlice
type MessageRespSlice []*MessageResp

func (m MessageRespSlice) Len() int { return len(m) }

func (m MessageRespSlice) Swap(i, j int) { m[i], m[j] = m[j], m[i] }

func (m MessageRespSlice) Less(i, j int) bool { return m[i].MessageSeq < m[j].MessageSeq }

// MessageResp 消息返回
type MessageResp struct {
	Header       MessageHeader `json:"header"`        // 消息头
	Setting      uint8         `json:"setting"`       // 设置
	MessageID    int64         `json:"message_id"`    // 服务端的消息ID(全局唯一)
	MessageIDStr string        `json:"message_idstr"` // 服务端的消息ID(全局唯一)
	ClientMsgNo  string        `json:"client_msg_no"` // 客户端消息唯一编号
	MessageSeq   uint32        `json:"message_seq"`   // 消息序列号 （用户唯一，有序递增）
	FromUID      string        `json:"from_uid"`      // 发送者UID
	ChannelID    string        `json:"channel_id"`    // 频道ID
	ChannelType  uint8         `json:"channel_type"`  // 频道类型
	Timestamp    int32         `json:"timestamp"`     // 服务器消息时间戳(10位，到秒)
	Payload      []byte        `json:"payload"`       // 消息内容
}

func (m *MessageResp) from(messageD *db.Message) {
	fm := lmproto.FramerFromUint8(messageD.Header)
	m.Header.NoPersist = util.BoolToInt(fm.NoPersist)
	m.Header.RedDot = util.BoolToInt(fm.RedDot)
	m.Header.SyncOnce = util.BoolToInt(fm.SyncOnce)
	m.Setting = messageD.Setting
	m.MessageID = messageD.MessageID
	m.MessageIDStr = strconv.FormatInt(messageD.MessageID, 10)
	m.ClientMsgNo = messageD.ClientMsgNo
	m.MessageSeq = messageD.MessageSeq
	m.FromUID = messageD.FromUID
	m.Timestamp = messageD.Timestamp

	realChannelID := messageD.ChannelID
	if messageD.ChannelType == ChannelTypePerson {
		if strings.Contains(messageD.ChannelID, "@") {
			channelIDs := strings.Split(messageD.ChannelID, "@")
			for _, channelID := range channelIDs {
				if messageD.FromUID != channelID {
					realChannelID = channelID
				}
			}
		}
	}
	m.ChannelID = realChannelID
	m.ChannelType = messageD.ChannelType
	m.Payload = messageD.Payload
}

// MessageHeader Message header
type MessageHeader struct {
	NoPersist int `json:"no_persist"` // Is it not persistent
	RedDot    int `json:"red_dot"`    // Whether to show red dot
	SyncOnce  int `json:"sync_once"`  // This message is only synchronized or consumed once
}

type clearConversationUnreadReq struct {
	UID         string `json:"uid"`
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
}

func (req clearConversationUnreadReq) Check() error {
	if req.UID == "" {
		return errors.New("uid cannot be empty")
	}
	if req.ChannelID == "" || req.ChannelType == 0 {
		return errors.New("channel_id or channel_type cannot be empty")
	}
	return nil
}

type deleteChannelReq struct {
	UIDS        []string `json:"uids"`
	ChannelID   string   `json:"channel_id"`
	ChannelType uint8    `json:"channel_type"`
}

func (req deleteChannelReq) Check() error {
	if len(req.UIDS) <= 0 {
		return errors.New("Uids cannot be empty")
	}
	if req.ChannelID == "" || req.ChannelType == 0 {
		return errors.New("channel_id or channel_type cannot be empty")
	}
	return nil
}

type syncUserConversationResp struct {
	ChannelID       string         `json:"channel_id"`         // 频道ID
	ChannelType     uint8          `json:"channel_type"`       // 频道类型
	Unread          int            `json:"unread"`             // 未读消息
	Timestamp       int64          `json:"timestamp"`          // 最后一次会话时间
	LastMsgSeq      uint32         `json:"last_msg_seq"`       // 最后一条消息seq
	LastClientMsgNo string         `json:"last_client_msg_no"` // 最后一次消息客户端编号
	OffsetMsgSeq    int64          `json:"offset_msg_seq"`     // 偏移位的消息seq
	Version         int64          `json:"version"`            // 数据版本
	Recents         []*MessageResp `json:"recents"`            // 最近N条消息
}

type channelRecentMessageReq struct {
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
	LastMsgSeq  uint32 `json:"last_msg_seq"`
}

type channelRecentMessage struct {
	ChannelID   string         `json:"channel_id"`
	ChannelType uint8          `json:"channel_type"`
	Messages    []*MessageResp `json:"messages"`
}

// MessageSendReq 消息发送请求
type MessageSendReq struct {
	Header      MessageHeader `json:"header"`        // 消息头
	ClientMsgNo string        `json:"client_msg_no"` // 客户端消息编号（相同编号，客户端只会显示一条）
	FromUID     string        `json:"from_uid"`      // 发送者UID
	ChannelID   string        `json:"channel_id"`    // 频道ID
	ChannelType uint8         `json:"channel_type"`  // 频道类型
	Subscribers []string      `json:"subscribers"`   // 订阅者 如果此字段有值，表示消息只发给指定的订阅者
	Payload     []byte        `json:"payload"`       // 消息内容
}

// Check 检查输入
func (m MessageSendReq) Check() error {
	if m.Payload == nil || len(m.Payload) <= 0 {
		return errors.New("payload不能为空！")
	}
	return nil
}

// ChannelInfoReq ChannelInfoReq
type ChannelInfoReq struct {
	ChannelID   string `json:"channel_id"`   // 频道ID
	ChannelType uint8  `json:"channel_type"` // 频道类型
	Bind        string `json:"bind"`         // 绑定的频道（如果有值标示 这个频道的成员将使用bind对应频道的成员）
	Forbidden   int    `json:"forbidden"`    // 是否禁言
}

// ChannelCreateReq 频道创建请求
type ChannelCreateReq struct {
	ChannelInfoReq
	Subscribers []string `json:"subscribers"` // 订阅者
}

// Check 检查请求参数
func (r ChannelCreateReq) Check() error {
	if strings.TrimSpace(r.ChannelID) == "" {
		return errors.New("频道ID不能为空！")
	}
	if r.ChannelType == 0 {
		return errors.New("频道类型错误！")
	}
	return nil
}

type subscriberAddReq struct {
	ChannelID   string   `json:"channel_id"`   // 频道ID
	ChannelType uint8    `json:"channel_type"` // 频道类型
	Reset       int      `json:"reset"`        // 是否重置订阅者 （0.不重置 1.重置），选择重置，将删除原来的所有成员
	Subscribers []string `json:"subscribers"`  // 订阅者
}

func (s subscriberAddReq) Check() error {
	if strings.TrimSpace(s.ChannelID) == "" {
		return errors.New("频道ID不能为空！")
	}
	if s.Subscribers == nil || len(s.Subscribers) <= 0 {
		return errors.New("订阅者不能为空！")
	}
	return nil
}

type subscriberRemoveReq struct {
	ChannelID   string   `json:"channel_id"`
	ChannelType uint8    `json:"channel_type"`
	Subscribers []string `json:"subscribers"`
}

func (s subscriberRemoveReq) Check() error {
	if strings.TrimSpace(s.ChannelID) == "" {
		return errors.New("频道ID不能为空！")
	}
	if s.Subscribers == nil || len(s.Subscribers) <= 0 {
		return errors.New("订阅者不能为空！")
	}
	return nil
}

type blacklistReq struct {
	ChannelID   string   `json:"channel_id"`   // 频道ID
	ChannelType uint8    `json:"channel_type"` // 频道类型
	UIDs        []string `json:"uids"`         // 订阅者
}

func (r blacklistReq) Check() error {
	if r.ChannelID == "" {
		return errors.New("channel_id不能为空！")
	}
	if r.ChannelType == 0 {
		return errors.New("频道类型不能为0！")
	}
	if len(r.UIDs) <= 0 {
		return errors.New("uids不能为空！")
	}
	return nil
}

// ChannelDeleteReq 删除频道请求
type ChannelDeleteReq struct {
	ChannelID   string `json:"channel_id"`   // 频道ID
	ChannelType uint8  `json:"channel_type"` // 频道类型
}

type whitelistReq struct {
	ChannelID   string   `json:"channel_id"`   // 频道ID
	ChannelType uint8    `json:"channel_type"` // 频道类型
	UIDs        []string `json:"uids"`         // 订阅者
}

func (r whitelistReq) Check() error {
	if r.ChannelID == "" {
		return errors.New("channel_id不能为空！")
	}
	if r.ChannelType == 0 {
		return errors.New("频道类型不能为0！")
	}
	if len(r.UIDs) <= 0 {
		return errors.New("uids不能为空！")
	}
	return nil
}

type syncReq struct {
	UID        string `json:"uid"`         // 用户uid
	MessageSeq uint32 `json:"message_seq"` // 客户端最大消息序列号
	Limit      int64  `json:"limit"`       // 消息数量限制
}

func (r syncReq) Check() error {
	if strings.TrimSpace(r.UID) == "" {
		return errors.New("用户uid不能为空！")
	}
	if r.MessageSeq < 0 {
		return errors.New("最大序消息列号不能为空！")
	}
	if r.Limit < 0 {
		return errors.New("limit不能为负数！")
	}
	return nil
}

type syncMessageResp struct {
	MinMessageSeq uint32         `json:"min_message_seq"` // 开始序列号
	MaxMessageSeq uint32         `json:"max_message_seq"` // 结束序列号
	More          int            `json:"more"`            // 是否还有更多 1.是 0.否
	Messages      []*MessageResp `json:"messages"`        // 消息数据
}

type syncackReq struct {
	// 用户uid
	UID string `json:"uid"`
	// 最后一次同步的message_seq
	LastMessageSeq uint32 `json:"last_message_seq"`
}

func (s syncackReq) Check() error {
	if strings.TrimSpace(s.UID) == "" {
		return errors.New("用户UID不能为空！")
	}
	if s.LastMessageSeq == 0 {
		return errors.New("最后一次messageSeq不能为0！")
	}
	return nil
}
