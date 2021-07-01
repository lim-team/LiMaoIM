package lim

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lim-team/LiMaoIM/internal/db"
	"github.com/lim-team/LiMaoIM/pkg/limlog"
	"github.com/lim-team/LiMaoIM/pkg/lmhttp"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"go.uber.org/zap"
)

// MessageAPI MessageAPI
type MessageAPI struct {
	l *LiMao
	limlog.Log
}

// NewMessageAPI NewMessageAPI
func NewMessageAPI(lim *LiMao) *MessageAPI {
	return &MessageAPI{
		l:   lim,
		Log: limlog.NewLIMLog("MessageApi"),
	}
}

// Route route
func (m *MessageAPI) Route(r *lmhttp.LMHttp) {
	r.POST("/message/send", m.send)
	r.POST("/message/sync", m.syncMessages)
}

func (m *MessageAPI) send(c *lmhttp.Context) {
	var req MessageSendReq
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	channelID := req.ChannelID
	channelType := req.ChannelType
	if strings.TrimSpace(channelID) == "" && len(req.Subscribers) > 0 { //如果没频道ID 但是有订阅者，则创建一个临时频道
		channelID = fmt.Sprintf("%s%s", util.GenUUID(), m.l.opts.TmpChannelSuffix)
		channelType = ChannelTypeGroup
		m.l.channelManager.CreateTmpChannel(channelID, channelType, req.Subscribers)
	}
	m.Debug("发送消息内容：", zap.String("msg", util.ToJSON(req)))
	if strings.TrimSpace(channelID) != "" { //指定了频道 正常发送
		err := m.sendMessageToChannel(req, channelID, channelType, req.ClientMsgNo)
		if err != nil {
			c.ResponseError(err)
			return
		}
	} else {
		m.Error("无法处理发送消息请求！", zap.Any("req", req))
		c.ResponseError(errors.New("无法处理发送消息请求！"))
		return
	}
	c.ResponseOK()
}

func (m *MessageAPI) sendMessageToChannel(req MessageSendReq, channelID string, channelType uint8, clientMsgNo string) error {
	var messageID = m.l.packetHandler.genMessageID()
	fakeChannelID := channelID
	if channelType == ChannelTypePerson && req.FromUID != "" {
		fakeChannelID = GetFakeChannelIDWith(req.FromUID, channelID)
	}
	// 获取频道
	channel, err := m.l.channelManager.GetChannel(fakeChannelID, channelType)
	if err != nil {
		m.Error("查询频道信息失败！", zap.Error(err))
		return errors.New("查询频道信息失败！")
	}
	if channel == nil {
		return errors.New("频道信息不存在！")
	}
	if strings.TrimSpace(clientMsgNo) == "" {
		clientMsgNo = util.GenUUID()
	}
	var messageSeq uint32
	if req.Header.NoPersist == 0 && req.Header.SyncOnce != 1 {
		messageSeq, err = m.l.store.GetNextMessageSeq(fakeChannelID, channelType)
		if err != nil {
			m.Error("获取频道消息序列号失败！", zap.String("channelID", fakeChannelID), zap.Uint8("channelType", channelType), zap.Error(err))
			return errors.New("获取频道消息序列号失败！")
		}
	}
	subscribers := req.Subscribers
	if len(subscribers) > 0 {
		subscribers = util.RemoveRepeatedElement(req.Subscribers)
	}
	msg := &Message{
		RecvPacket: lmproto.RecvPacket{
			Framer: lmproto.Framer{
				RedDot:    util.IntToBool(req.Header.RedDot),
				SyncOnce:  util.IntToBool(req.Header.SyncOnce),
				NoPersist: util.IntToBool(req.Header.NoPersist),
			},
			MessageID:   messageID,
			ClientMsgNo: clientMsgNo,
			FromUID:     req.FromUID,
			MessageSeq:  messageSeq,
			ChannelID:   channelID,
			ChannelType: channelType,
			Timestamp:   int32(time.Now().Unix()),
			Payload:     req.Payload,
		},
		fromDeviceFlag: lmproto.SYSTEM,
		Subscribers:    subscribers,
	}
	if !msg.NoPersist && !msg.SyncOnce {
		_, err = m.l.store.AppendMessage(&db.Message{
			Header:      lmproto.ToFixHeaderUint8(msg),
			Setting:     msg.Setting.ToUint8(),
			MessageID:   msg.MessageID,
			MessageSeq:  messageSeq,
			ClientMsgNo: msg.ClientMsgNo,
			Timestamp:   msg.Timestamp,
			FromUID:     msg.FromUID,
			ChannelID:   fakeChannelID,
			ChannelType: msg.ChannelType,
			Payload:     msg.Payload,
		})
		if err != nil {
			m.Error("Failed to save history message", zap.Error(err))
			return errors.New("Failed to save history message")
		}
	}
	// 将消息放入频道
	err = channel.PutMessage(msg)
	if err != nil {
		m.Error("将消息放入频道内失败！", zap.Error(err))
		return errors.New("将消息放入频道内失败！")
	}
	return nil
}

func (m *MessageAPI) syncMessages(c *lmhttp.Context) {
	var req struct {
		UID              string `json:"uid"` // 当前登录用户的uid
		ChannelID        string `json:"channel_id"`
		ChannelType      uint8  `json:"channel_type"`
		OffsetMessageSeq uint32 `json:"offset_message_seq"` // 偏移序号
		EndMessageSeq    uint32 `json:"end_message_seq"`    // 结束偏移量
		Limit            int    `json:"limit"`              // 每次同步数量限制
		Reverse          int    `json:"reverse"`            // 是否反转查询
	}
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}

	fakeChannelID := req.ChannelID
	if req.ChannelType == ChannelTypePerson {
		fakeChannelID = GetFakeChannelIDWith(req.UID, req.ChannelID)
	}
	messages, err := m.l.store.GetMessagesWithOptions(fakeChannelID, req.ChannelType, req.OffsetMessageSeq, uint64(req.Limit), req.Reverse == 1, req.EndMessageSeq)
	if err != nil {
		c.ResponseError(err)
		return
	}
	messageResps := make([]*MessageResp, 0, len(messages))
	if len(messages) > 0 {
		for _, message := range messages {
			messageResp := &MessageResp{}
			messageResp.from(message)
			messageResps = append(messageResps, messageResp)
		}
	}
	c.JSON(http.StatusOK, messageResps)
}
