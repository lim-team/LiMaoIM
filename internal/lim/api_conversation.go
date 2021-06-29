package lim

import (
	"errors"
	"net/http"
	"strings"

	"github.com/lim-team/LiMaoIM/pkg/limlog"
	"github.com/lim-team/LiMaoIM/pkg/lmhttp"
	"go.uber.org/zap"
)

// ConversationAPI ConversationAPI
type ConversationAPI struct {
	l *LiMao
	limlog.Log
}

// NewConversationAPI NewConversationAPI
func NewConversationAPI(lim *LiMao) *ConversationAPI {
	return &ConversationAPI{
		l:   lim,
		Log: limlog.NewLIMLog("ConversationAPI"),
	}
}

// Route 路由
func (s *ConversationAPI) Route(r *lmhttp.LMHttp) {
	r.GET("/conversations", s.conversationsList)
	r.POST("/conversations/clearUnread", s.clearConversationUnread)
	r.POST("/conversations/setUnread", s.setConversationUnread)
	r.POST("/conversations/delete", s.deleteConversation)
	r.POST("/conversation/sync", s.syncUserConversation)
	r.POST("/conversation/syncMessages", s.syncRecentMessages)
}

// Get a list of recent conversations
func (s *ConversationAPI) conversationsList(c *lmhttp.Context) {
	uid := c.Query("uid")
	if strings.TrimSpace(uid) == "" {
		c.ResponseError(errors.New("uid cannot be empty"))
		return
	}
	conversations := s.l.conversationManager.GetConversations(uid, 0)
	conversationResps := make([]conversationResp, 0)
	if conversations != nil {
		for _, conversation := range conversations {
			fakeChannelID := conversation.ChannelID
			if conversation.ChannelType == ChannelTypePerson {
				fakeChannelID = GetFakeChannelIDWith(uid, conversation.ChannelID)
			}
			// 获取到偏移位内的指定最大条数的最新消息
			message, err := s.l.store.GetMessage(fakeChannelID, conversation.ChannelType, conversation.LastMsgSeq)
			if err != nil {
				s.Error("Failed to query recent news", zap.Error(err))
				c.ResponseError(err)
				return
			}
			messageResp := &MessageResp{}
			if message != nil {
				messageResp.from(message)
			}
			conversationResps = append(conversationResps, conversationResp{
				ChannelID:   conversation.ChannelID,
				ChannelType: conversation.ChannelType,
				Unread:      conversation.UnreadCount,
				Timestamp:   conversation.Timestamp,
				LastMessage: messageResp,
			})
		}
	}
	c.JSON(http.StatusOK, conversationResps)
}

// 清楚会话未读数量
func (s *ConversationAPI) clearConversationUnread(c *lmhttp.Context) {
	var req clearConversationUnreadReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(err)
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	err := s.l.conversationManager.SetConversationUnread(req.UID, req.ChannelID, req.ChannelType, 0)
	if err != nil {
		c.ResponseError(err)
		return
	}
	c.ResponseOK()
}

func (s *ConversationAPI) setConversationUnread(c *lmhttp.Context) {
	var req struct {
		UID         string `json:"uid"`
		ChannelID   string `json:"channel_id"`
		ChannelType uint8  `json:"channel_type"`
		Unread      int    `json:"unread"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(err)
		return
	}
	if req.UID == "" {
		c.ResponseError(errors.New("UID cannot be empty"))
		return
	}
	if req.ChannelID == "" || req.ChannelType == 0 {
		c.ResponseError(errors.New("channel_id or channel_type cannot be empty"))
		return
	}
	err := s.l.conversationManager.SetConversationUnread(req.UID, req.ChannelID, req.ChannelType, req.Unread)
	if err != nil {
		c.ResponseError(err)
		return
	}
	c.ResponseOK()
}

func (s *ConversationAPI) deleteConversation(c *lmhttp.Context) {
	var req deleteChannelReq
	if err := c.BindJSON(&req); err != nil {
		s.Error("Data Format", zap.Error(err))
		c.ResponseError(err)
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	// 删除最近会话
	err := s.l.conversationManager.DeleteConversation(req.UIDS, req.ChannelID, req.ChannelType)
	if err != nil {
		s.Error("删除最近会话！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	c.ResponseOK()
}

func (s *ConversationAPI) syncUserConversation(c *lmhttp.Context) {
	var req struct {
		UID         string `json:"uid"`
		Version     int64  `json:"version"`       // 当前客户端的会话最大版本号(客户端最新会话的时间戳)
		LastMsgSeqs string `json:"last_msg_seqs"` // 客户端所有会话的最后一条消息序列号 格式： channelID:channelType:last_msg_seq|channelID:channelType:last_msg_seq
		MsgCount    int64  `json:"msg_count"`     // 每个会话消息数量
	}
	if err := c.BindJSON(&req); err != nil {
		s.Error("格式有误！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	msgCount := req.MsgCount
	if msgCount == 0 {
		msgCount = 100
	}
	conversations := s.l.conversationManager.GetConversations(req.UID, req.Version)

	resps := make([]syncUserConversationResp, 0, len(conversations))
	if len(conversations) > 0 {
		for _, conversation := range conversations {
			syncUserConversationR := syncUserConversationResp{
				ChannelID:       conversation.ChannelID,
				ChannelType:     conversation.ChannelType,
				Unread:          conversation.UnreadCount,
				Timestamp:       conversation.Timestamp,
				LastMsgSeq:      conversation.LastMsgSeq,
				LastClientMsgNo: conversation.LastClientMsgNo,
				Version:         conversation.Version,
			}
			resps = append(resps, syncUserConversationR)
		}
	}
	c.JSON(http.StatusOK, resps)
}

func (s *ConversationAPI) syncRecentMessages(c *lmhttp.Context) {
	var req struct {
		UID      string                     `json:"uid"`
		Channels []*channelRecentMessageReq `json:"channels"`
		MsgCount int                        `json:"msg_count"`
	}
	if err := c.BindJSON(&req); err != nil {
		s.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	msgCount := req.MsgCount
	if msgCount <= 0 {
		msgCount = 15
	}
	channelRecentMessages := make([]*channelRecentMessage, 0)
	if len(req.Channels) > 0 {
		for _, channel := range req.Channels {
			fakeChannelID := channel.ChannelID
			if channel.ChannelType == ChannelTypePerson {
				fakeChannelID = GetFakeChannelIDWith(req.UID, channel.ChannelID)
			}
			recentMessages, err := s.l.store.GetMessages(fakeChannelID, channel.ChannelType, channel.LastMsgSeq, uint64(msgCount))
			if err != nil {
				s.Error("查询最近消息失败！", zap.Error(err))
				c.ResponseError(err)
				return
			}
			messageResps := make([]*MessageResp, 0, len(recentMessages))
			if len(recentMessages) > 0 {
				for _, recentMessage := range recentMessages {
					messageResp := &MessageResp{}
					messageResp.from(recentMessage)
					messageResps = append(messageResps, messageResp)
				}
			}
			channelRecentMessages = append(channelRecentMessages, &channelRecentMessage{
				ChannelID:   channel.ChannelID,
				ChannelType: channel.ChannelType,
				Messages:    messageResps,
			})
		}
	}
	c.JSON(http.StatusOK, channelRecentMessages)
}
