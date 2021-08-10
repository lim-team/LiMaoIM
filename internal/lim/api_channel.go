package lim

import (
	"net/http"
	"strings"
	"sync"

	"github.com/lim-team/LiMaoIM/pkg/limlog"
	"github.com/lim-team/LiMaoIM/pkg/lmhttp"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// ChannelAPI ChannelAPI
type ChannelAPI struct {
	l              *LiMao
	subscriberLock sync.RWMutex
	limlog.Log
}

// NewChannelAPI 创建API
func NewChannelAPI(lim *LiMao) *ChannelAPI {
	return &ChannelAPI{
		Log: limlog.NewLIMLog("ChannelAPI"),
		l:   lim,
	}
}

// Route Route
func (ch *ChannelAPI) Route(r *lmhttp.LMHttp) {
	r.POST("/channel", ch.channelCreateOrUpdate)
	r.POST("/channel/subscriber_add", ch.addSubscriber)
	r.POST("/channel/subscriber_remove", ch.removeSubscriber)
	// 删除频道
	r.POST("/channel/delete", ch.channelDelete)
	// 频道黑名单
	r.POST("/channel/blacklist_add", ch.blacklistAdd) // 添加白明单
	r.POST("/channel/blacklist_set", ch.blacklistSet) // 设置黑明单（覆盖原来的黑名单数据）
	r.POST("/channel/blacklist_remove", ch.blacklistRemove)

	// 白名单
	r.POST("/channel/whitelist_add", ch.whitelistAdd)
	r.POST("/channel/whitelist_set", ch.whitelistSet) // 设置白明单（覆盖
	r.POST("/channel/whitelist_remove", ch.whitelistRemove)

	// 同步频道消息
	r.POST("/channel/messagesync", ch.syncMessages)
}

func (ch *ChannelAPI) channelCreateOrUpdate(c *lmhttp.Context) {
	var req ChannelCreateReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.Wrap(err, "数据格式有误！"))
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	if req.ChannelType == ChannelTypePerson {
		c.ResponseError(errors.New("暂不支持个人频道！"))
		return
	}
	channelInfo := NewChannelInfo(req.ChannelID, req.ChannelType)
	channelInfo.Parent = req.Bind

	err := ch.l.store.AddOrUpdateChannel(channelInfo)
	if err != nil {
		c.ResponseError(err)
		ch.Error("创建频道失败！", zap.Error(err))
		return
	}
	err = ch.l.store.RemoveAllSubscriber(req.ChannelID, req.ChannelType)
	if err != nil {
		ch.Error("移除所有订阅者失败！", zap.Error(err))
		c.ResponseError(errors.New("移除所有订阅者失败！"))
		return
	}
	err = ch.l.store.AddSubscribers(req.ChannelID, req.ChannelType, req.Subscribers)
	if err != nil {
		ch.Error("添加订阅者失败！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	ch.l.channelManager.DeleteChannelFromCache(req.ChannelID, req.ChannelType)
	c.ResponseOK()
}

func (ch *ChannelAPI) addSubscriber(c *lmhttp.Context) {
	var req subscriberAddReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.Wrap(err, "添加订阅者失败！"))
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	if req.ChannelType == ChannelTypePerson {
		c.ResponseError(errors.New("个人频道不支持添加订阅者！"))
		return
	}
	if req.ChannelType == 0 {
		req.ChannelType = ChannelTypeGroup //默认为群
	}
	channel, err := ch.l.channelManager.GetChannel(req.ChannelID, req.ChannelType)
	if err != nil {
		ch.Error("获取频道失败！", zap.String("channel", req.ChannelID), zap.Error(err))
		c.ResponseError(errors.Wrap(err, "获取频道失败！"))
		return
	}
	if channel == nil {
		ch.Error("频道不存在！", zap.String("channel_id", req.ChannelID), zap.Uint8("channel_type", req.ChannelType))
		c.ResponseError(errors.New("频道并不存在！"))
		return
	}
	existSubscribers := make([]string, 0)
	if req.Reset == 1 {
		err = ch.l.store.RemoveAllSubscriber(req.ChannelID, req.ChannelType)
		if err != nil {
			ch.Error("移除所有订阅者失败！", zap.Error(err))
			c.ResponseError(err)
			return
		}
		channel.RemoveAllSubscriber()
	} else {

		existSubscribers, err = ch.l.store.GetSubscribers(req.ChannelID, req.ChannelType)
		if err != nil {
			ch.Error("获取所有订阅者失败！", zap.Error(err))
			c.ResponseError(err)
			return
		}
	}

	newSubscribers := make([]string, 0, len(req.Subscribers))
	for _, subscriber := range req.Subscribers {
		if !util.ArrayContains(existSubscribers, subscriber) {
			newSubscribers = append(newSubscribers, subscriber)
		}
	}
	if len(newSubscribers) > 0 {
		err = ch.l.store.AddSubscribers(req.ChannelID, req.ChannelType, newSubscribers)
		if err != nil {
			ch.Error("添加订阅者失败！", zap.Error(err))
			c.ResponseError(err)
			return
		}
	}
	c.ResponseOK()
}

func (ch *ChannelAPI) removeSubscriber(c *lmhttp.Context) {
	var req subscriberRemoveReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.Wrap(err, "数据格式有误！"))
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	channel, err := ch.l.channelManager.GetChannel(req.ChannelID, req.ChannelType)
	if err != nil {
		ch.Error("获取频道失败！", zap.Error(err), zap.String("channelId", req.ChannelID))
		c.ResponseError(errors.Wrap(err, "获取频道失败！"))
		return
	}

	err = ch.l.store.RemoveSubscribers(req.ChannelID, req.ChannelType, req.Subscribers)
	if err != nil {
		ch.Error("移除订阅者失败！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	for _, subscriber := range req.Subscribers {
		channel.RemoveSubscriber(subscriber)
	}
	err = ch.l.conversationManager.DeleteConversation(req.Subscribers, req.ChannelID, req.ChannelType)
	if err != nil {
		ch.Error("删除最近会话失败！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	c.ResponseOK()
}

func (ch *ChannelAPI) blacklistAdd(c *lmhttp.Context) {
	var req blacklistReq
	if err := c.BindJSON(&req); err != nil {
		ch.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	err := ch.l.store.AddDenylist(req.ChannelID, req.ChannelType, req.UIDs)
	if err != nil {
		ch.Error("添加黑名单失败！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	// 增加到缓存中
	channelObj, err := ch.l.channelManager.GetChannel(req.ChannelID, req.ChannelType)
	if err != nil {
		c.ResponseError(err)
		return
	}
	channelObj.AddDenylist(req.UIDs)

	c.ResponseOK()
}

func (ch *ChannelAPI) blacklistSet(c *lmhttp.Context) {
	var req blacklistReq
	if err := c.BindJSON(&req); err != nil {
		ch.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	if strings.TrimSpace(req.ChannelID) == "" {
		c.ResponseError(errors.New("频道ID不能为空！"))
		return
	}
	err := ch.l.store.RemoveAllDenylist(req.ChannelID, req.ChannelType)
	if err != nil {
		ch.Error("移除所有黑明单失败！", zap.Error(err))
		c.ResponseError(errors.New("移除所有黑明单失败！"))
		return
	}
	if len(req.UIDs) > 0 {
		err := ch.l.store.AddDenylist(req.ChannelID, req.ChannelType, req.UIDs)
		if err != nil {
			ch.Error("添加黑名单失败！", zap.Error(err))
			c.ResponseError(err)
			return
		}
	}
	// 增加到缓存中
	channelObj, err := ch.l.channelManager.GetChannel(req.ChannelID, req.ChannelType)
	if err != nil {
		c.ResponseError(err)
		return
	}
	channelObj.SetDenylist(req.UIDs)

	c.ResponseOK()
}

func (ch *ChannelAPI) blacklistRemove(c *lmhttp.Context) {
	var req blacklistReq
	if err := c.BindJSON(&req); err != nil {
		ch.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	err := ch.l.store.RemoveDenylist(req.ChannelID, req.ChannelType, req.UIDs)
	if err != nil {
		ch.Error("移除黑名单失败！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	// 缓存中移除
	channelObj, err := ch.l.channelManager.GetChannel(req.ChannelID, req.ChannelType)
	if err != nil {
		c.ResponseError(err)
		return
	}
	channelObj.RemoveDenylist(req.UIDs)
	c.ResponseOK()
}

// 删除频道
func (ch *ChannelAPI) channelDelete(c *lmhttp.Context) {
	var req ChannelDeleteReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.Wrap(err, "数据格式有误！"))
		return
	}

	err := ch.l.store.DeleteChannelAndClearMessages(req.ChannelID, req.ChannelType)
	if err != nil {
		c.ResponseError(err)
		return
	}

	ch.l.channelManager.DeleteChannel(req.ChannelID, req.ChannelType)
	c.ResponseOK()
}

// ----------- 白名单 -----------

// 添加白名单
func (ch *ChannelAPI) whitelistAdd(c *lmhttp.Context) {
	var req whitelistReq
	if err := c.BindJSON(&req); err != nil {
		ch.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	err := ch.l.store.AddAllowlist(req.ChannelID, req.ChannelType, req.UIDs)
	if err != nil {
		ch.Error("添加白名单失败！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	// 增加到缓存中
	channelObj, err := ch.l.channelManager.GetChannel(req.ChannelID, req.ChannelType)
	if err != nil {
		c.ResponseError(err)
		return
	}
	channelObj.AddAllowlist(req.UIDs)

	c.ResponseOK()
}
func (ch *ChannelAPI) whitelistSet(c *lmhttp.Context) {
	var req whitelistReq
	if err := c.BindJSON(&req); err != nil {
		ch.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	if strings.TrimSpace(req.ChannelID) == "" {
		c.ResponseError(errors.New("频道ID不能为空！"))
		return
	}
	err := ch.l.store.RemoveAllAllowlist(req.ChannelID, req.ChannelType)
	if err != nil {
		ch.Error("移除所有白明单失败！", zap.Error(err))
		c.ResponseError(errors.New("移除所有白明单失败！"))
		return
	}
	if len(req.UIDs) > 0 {
		err := ch.l.store.AddAllowlist(req.ChannelID, req.ChannelType, req.UIDs)
		if err != nil {
			ch.Error("添加白名单失败！", zap.Error(err))
			c.ResponseError(err)
			return
		}
	}
	// 增加到缓存中
	channelObj, err := ch.l.channelManager.GetChannel(req.ChannelID, req.ChannelType)
	if err != nil {
		c.ResponseError(err)
		return
	}
	channelObj.SetAllowlist(req.UIDs)

	c.ResponseOK()
}

// 移除白名单
func (ch *ChannelAPI) whitelistRemove(c *lmhttp.Context) {
	var req whitelistReq
	if err := c.BindJSON(&req); err != nil {
		ch.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	err := ch.l.store.RemoveAllowlist(req.ChannelID, req.ChannelType, req.UIDs)
	if err != nil {
		ch.Error("移除白名单失败！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	// 缓存中移除
	channelObj, err := ch.l.channelManager.GetChannel(req.ChannelID, req.ChannelType)
	if err != nil {
		c.ResponseError(err)
		return
	}
	channelObj.RemoveAllowlist(req.UIDs)
	c.ResponseOK()
}

// 同步频道内的消息
func (ch *ChannelAPI) syncMessages(c *lmhttp.Context) {
	var req struct {
		LoginUID      string `json:"login_uid"` // 当前登录用户的uid
		ChannelID     string `json:"channel_id"`
		ChannelType   uint8  `json:"channel_type"`
		MinMessageSeq uint32 `json:"min_message_seq"` // 最小序列号
		MaxMessageSeq uint32 `json:"max_message_seq"` // 最大序列号
		Limit         int    `json:"limit"`           // 每次同步数量限制
		Reverse       int    `json:"reverse"`         // 是否反转查询 true: 上拉 messageSeq从小到大 false: 下拉 messageSeq从大到小
	}
	if err := c.BindJSON(&req); err != nil {
		ch.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}

	fakeChannelID := req.ChannelID
	if req.ChannelType == ChannelTypePerson {
		fakeChannelID = GetFakeChannelIDWith(req.LoginUID, req.ChannelID)
	}
	messages, err := ch.l.store.GetMessagesWithOptions(fakeChannelID, req.ChannelType, req.MinMessageSeq, uint64(req.Limit), req.Reverse == 1, req.MaxMessageSeq)
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
	var more bool = true // 是否有更多数据
	if len(messageResps) < req.Limit {
		more = false
	}
	if len(messageResps) > 0 {
		messageSeq := messageResps[len(messageResps)-1].MessageSeq
		if req.Reverse == 1 {
			if req.MinMessageSeq != 0 {
				if req.MinMessageSeq == messageSeq {
					more = false
				}
			}
		} else {
			if req.MaxMessageSeq != 0 {
				if req.MaxMessageSeq == messageSeq {
					more = false
				}
			}
		}
	}
	c.JSON(http.StatusOK, syncMessageResp{
		MinMessageSeq: req.MinMessageSeq,
		MaxMessageSeq: req.MaxMessageSeq,
		More:          util.BoolToInt(more),
		Messages:      messageResps,
	})
}
