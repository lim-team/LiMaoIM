package lim

import (
	"fmt"
	"sync"

	"github.com/lim-team/LiMaoIM/internal/db"
	"github.com/lim-team/LiMaoIM/internal/lim/rpc"
	"github.com/lim-team/LiMaoIM/pkg/limlog"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/lmproxyproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"go.uber.org/zap"
)

// ChannelInfo ChannelInfo
type ChannelInfo struct {
	ChannelID   string `json:"-"`
	ChannelType uint8  `json:"-"`
	Mute        bool   `json:"mute"`   // Whether to mute
	Parent      string `json:"parent"` // Parent channel
}

// ToMap ToMap
func (c *ChannelInfo) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"mute":   c.Mute,
		"parent": c.Parent,
	}
}

func (c *ChannelInfo) from(mp map[string]interface{}) {
	if mp["mute"] != nil {
		c.Mute = mp["mute"].(bool)
	}
	if mp["parent"] != nil {
		c.Parent = mp["parent"].(string)
	}

}

// NewChannelInfo NewChannelInfo
func NewChannelInfo(channelID string, channelType uint8) *ChannelInfo {
	return &ChannelInfo{
		ChannelID:   channelID,
		ChannelType: channelType,
	}
}

// Channel Channel
type Channel struct {
	*ChannelInfo
	blacklist     sync.Map // 黑名单
	whitelist     sync.Map // 白名单
	subscriberMap sync.Map // 订阅者
	l             *LiMao
	limlog.Log
	MessageChan chan *Message
}

// NewChannel NewChannel
func NewChannel(channelInfo *ChannelInfo, l *LiMao) *Channel {

	return &Channel{
		ChannelInfo:   channelInfo,
		blacklist:     sync.Map{},
		whitelist:     sync.Map{},
		subscriberMap: sync.Map{},
		MessageChan:   make(chan *Message),
		l:             l,
		Log:           limlog.NewLIMLog(fmt.Sprintf("channel[%s-%d]", channelInfo.ChannelID, channelInfo.ChannelType)),
	}
}

// LoadData load data
func (c *Channel) LoadData() error {

	if c.ChannelType == ChannelTypePerson && c.l.opts.IsFakeChannel(c.ChannelID) {
		return nil
	}
	if err := c.l.systemUIDManager.LoadIfNeed(); err != nil { // 加载系统账号
		return err
	}
	if err := c.initSubscribers(); err != nil { // 初始化订阅者
		return err
	}
	if err := c.initBlacklist(); err != nil { // 初始化黑名单
		return err
	}
	if err := c.initWhitelist(); err != nil { // 初始化白名单
		return err
	}
	return nil
}

// 初始化订阅者
func (c *Channel) initSubscribers() error {
	if c.l.opts.HasDatasource() && c.ChannelType != ChannelTypePerson {
		subscribers, err := c.l.datasource.GetSubscribers(c.ChannelID, c.ChannelType)
		if err != nil {
			c.Error("从数据源获取频道订阅者失败！", zap.Error(err))
			return err
		}
		if len(subscribers) > 0 {
			for _, subscriber := range subscribers {
				c.subscriberMap.Store(subscriber, true)
			}
		}
	} else {
		// ---------- 订阅者  ----------
		subscribers, err := c.l.store.GetSubscribers(c.ChannelID, c.ChannelType)
		if err != nil {
			c.Error("获取频道订阅者失败！", zap.Error(err))
			return err
		}
		if len(subscribers) > 0 {
			for _, subscriber := range subscribers {
				c.subscriberMap.Store(subscriber, true)
			}
		}
	}
	return nil
}

// 初始化黑名单
func (c *Channel) initBlacklist() error {
	var blacklists []string
	var err error
	if c.l.opts.HasDatasource() {
		blacklists, err = c.l.datasource.GetBlacklist(c.ChannelID, c.ChannelType)
		if err != nil {
			c.Error("从数据源获取黑名单失败！", zap.Error(err))
			return err
		}
	} else {
		blacklists, err = c.l.store.GetDenylist(c.ChannelID, c.ChannelType)
		if err != nil {
			c.Error("获取黑名单失败！", zap.Error(err))
			return err
		}
	}
	if len(blacklists) > 0 {
		for _, uid := range blacklists {
			c.blacklist.Store(uid, true)
		}
	}
	return nil
}

// 初始化黑名单
func (c *Channel) initWhitelist() error {
	var whitelists []string
	var err error
	if c.l.opts.HasDatasource() {
		whitelists, err = c.l.datasource.GetWhitelist(c.ChannelID, c.ChannelType)
		if err != nil {
			c.Error("从数据源获取白名单失败！", zap.Error(err))
			return err
		}
	} else {
		whitelists, err = c.l.store.GetAllowlist(c.ChannelID, c.ChannelType)
		if err != nil {
			c.Error("获取白名单失败！", zap.Error(err))
			return err
		}
	}
	if len(whitelists) > 0 {
		for _, uid := range whitelists {
			c.whitelist.Store(uid, true)
		}
	}
	return nil
}

// ---------- 订阅者 ----------

// IsSubscriber 是否已订阅
func (c *Channel) IsSubscriber(uid string) bool {
	_, ok := c.subscriberMap.Load(uid)
	if ok {
		return ok
	}
	if len(c.Parent) > 0 { // TODO: 这里暂时只有设置了bind的都是true
		return true
	}
	return ok
}

// ---------- 黑名单 (怕怕😱) ----------

// IsBlacklist 是否在黑名单内
func (c *Channel) IsBlacklist(uid string) bool {
	_, ok := c.blacklist.Load(uid)
	return ok
}

// AddSubscriber Add subscribers
func (c *Channel) AddSubscriber(uid string) {
	c.subscriberMap.Store(uid, true)
}

// Allow Whether to allow sending of messages If it is in the white list or not in the black list, it is allowed to send
func (c *Channel) Allow(uid string) bool {

	systemUID := c.l.systemUIDManager.SystemUID(uid) // Is it a system account?
	if systemUID {
		return true
	}

	whitelistLength := 0
	c.whitelist.Range(func(_, _ interface{}) bool {
		whitelistLength++
		return false
	})
	if whitelistLength > 0 { // 如果白名单有内容，则只判断白名单
		_, ok := c.whitelist.Load(uid)
		return ok
	}
	return !c.IsBlacklist(uid)
}

// PutMessage put message
func (c *Channel) PutMessage(m *Message) error {
	// 组合订阅者
	subscribers := make([]string, 0) // TODO: 此处可以用对象pool来管理
	if len(m.Subscribers) > 0 {      // 如果指定了订阅者则消息只发给指定的订阅者，不将发送给其他订阅者
		subscribers = append(subscribers, m.Subscribers...)
	} else { // 默认将消息发送给频道的订阅者
		subscribers = append(subscribers, c.GetAllSubscribers()...)
	}
	c.Debug("subscribers", zap.Any("subscribers", subscribers))

	if c.l.opts.IsCluster {
		// 计算订阅者所在节点，属于本节点的，就去做存储和投递的逻辑，不是本节点的订阅者则投递到订阅者所在节点
		selfNodeSubscriber, otherNodeSubscriberList := c.calNodeSubscribers(subscribers)
		if len(otherNodeSubscriberList) > 0 {
			// 投递所属其他节点的消息
			recvPacketData, _ := c.l.protocol.EncodePacket(&m.RecvPacket, lmproto.LatestVersion)
			for _, nodeSubscribers := range otherNodeSubscriberList {
				req := &rpc.ForwardRecvPacketReq{
					No:             util.GenUUID(),
					Users:          nodeSubscribers.subscribers,
					Message:        recvPacketData,
					FromDeviceFlag: int32(m.fromDeviceFlag),
				}
				// 开始投递节点数据
				c.l.startDeliveryNodeData(&NodeInFlightData{
					NodeInFlightDataModel: db.NodeInFlightDataModel{
						No:     util.GenUUID(),
						NodeID: nodeSubscribers.node.NodeID,
						Req:    req,
					},
				})
			}
		}
		if selfNodeSubscriber != nil && len(selfNodeSubscriber.subscribers) > 0 {
			err := c.l.handleLocalSubscribersMessage(m, selfNodeSubscriber.subscribers)
			if err != nil {
				c.Error("处理本地订阅者失败！", zap.Error(err))
				return err
			}
		}
	} else {
		c.l.handleLocalSubscribersMessage(m, subscribers)
	}
	return nil
}

// GetAllSubscribers 获取所有订阅者
func (c *Channel) GetAllSubscribers() []string {
	subscribers := make([]string, 0)
	c.subscriberMap.Range(func(key, value interface{}) bool {
		subscribers = append(subscribers, key.(string))
		return true
	})
	return subscribers
}

// RemoveAllSubscriber 移除所有订阅者
func (c *Channel) RemoveAllSubscriber() {
	c.subscriberMap.Range(func(key, value interface{}) bool {
		c.subscriberMap.Delete(key)
		return true
	})
}

// RemoveSubscriber  移除订阅者
func (c *Channel) RemoveSubscriber(uid string) {
	c.subscriberMap.Delete(uid)
}

// AddDenylist 添加黑名单
func (c *Channel) AddDenylist(uids []string) {
	if len(uids) == 0 {
		return
	}
	for _, uid := range uids {
		c.blacklist.Store(uid, true)
	}
}

// SetDenylist SetDenylist
func (c *Channel) SetDenylist(uids []string) {
	c.blacklist.Range(func(key interface{}, value interface{}) bool {
		c.blacklist.Delete(key)
		return true
	})
	c.AddDenylist(uids)
}

// RemoveDenylist 移除黑名单
func (c *Channel) RemoveDenylist(uids []string) {
	if len(uids) == 0 {
		return
	}
	for _, uid := range uids {
		c.blacklist.Delete(uid)
	}
}

// ---------- 白名单 ----------

// AddAllowlist 添加白名单
func (c *Channel) AddAllowlist(uids []string) {
	if len(uids) == 0 {
		return
	}
	for _, uid := range uids {
		c.whitelist.Store(uid, true)
	}

}

// SetAllowlist SetAllowlist
func (c *Channel) SetAllowlist(uids []string) {
	c.whitelist.Range(func(key interface{}, value interface{}) bool {
		c.whitelist.Delete(key)
		return true
	})
	c.AddDenylist(uids)
}

// RemoveAllowlist 移除白名单
func (c *Channel) RemoveAllowlist(uids []string) {
	if len(uids) == 0 {
		return
	}
	for _, uid := range uids {
		c.whitelist.Delete(uid)
	}
}

// NodeSubscribers 节点对应的订阅者
type nodeSubscribers struct {
	node        *lmproxyproto.Node
	subscribers []string
}

// calSubscribersNodes 计算订阅者所在节点 TODO: 这里需要用对象池管理下
func (c *Channel) calNodeSubscribers(subscribers []string) (localNode *nodeSubscribers, othersNodeSubbscribers []*nodeSubscribers) {
	othersNodeSubbscribers = make([]*nodeSubscribers, 0, len(subscribers))
	for _, subscriber := range subscribers {
		nodeConfig := c.l.clusterManager.GetNode(subscriber)
		if nodeConfig.NodeID == c.l.opts.NodeID {
			if localNode == nil {
				localNode = &nodeSubscribers{
					node:        nodeConfig,
					subscribers: []string{subscriber},
				}
			} else {
				localNode.subscribers = append(localNode.subscribers, subscriber)
			}
			continue
		}

		var existNodeSub *nodeSubscribers
		for _, nodeSub := range othersNodeSubbscribers {
			if nodeSub.node.NodeID == nodeConfig.NodeID {
				existNodeSub = nodeSub
				break
			}
		}
		if existNodeSub != nil {
			existNodeSub.subscribers = append(existNodeSub.subscribers, subscriber)
		} else {
			othersNodeSubbscribers = append(othersNodeSubbscribers, &nodeSubscribers{
				node:        nodeConfig,
				subscribers: []string{subscriber},
			})
		}
	}
	return
}
