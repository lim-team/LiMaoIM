package lim

import (
	"fmt"
	"strings"
	"sync"

	"github.com/lim-team/LiMaoIM/pkg/util"
	"github.com/pkg/errors"
	"github.com/tangtaoit/limnet/pkg/limlog"
)

// ---------- 频道管理 ----------

// ChannelManager 频道管理
type ChannelManager struct {
	l                *LiMao
	channelMap       sync.Map
	tmpChannelMap    sync.Map // 临时频道 TODO: 这里存在运行时间长内存递增的问题，需要释放掉一些不需要的临时频道
	personChannelMap sync.Map // 临时频道 TODO: 同上
	limlog.Log
}

// NewChannelManager 创建一个频道管理者
func NewChannelManager(l *LiMao) *ChannelManager {
	return &ChannelManager{
		channelMap:       sync.Map{},
		tmpChannelMap:    sync.Map{},
		personChannelMap: sync.Map{},
		l:                l,
		Log:              limlog.NewLIMLog("ChannelManager"),
	}
}

// GetChannel 获取频道
func (cm *ChannelManager) GetChannel(channelID string, channelType uint8) (*Channel, error) {

	if strings.HasSuffix(channelID, cm.l.opts.TmpChannelSuffix) {
		return cm.GetTmpChannel(channelID, channelType)
	}
	if channelType == ChannelTypePerson {
		return cm.GetPersonChannel(channelID, channelType), nil
	}

	key := fmt.Sprintf("%s-%d", channelID, channelType)
	channelObj, _ := cm.channelMap.Load(key)
	if channelObj != nil {
		return channelObj.(*Channel), nil
	}
	channelInfo, err := cm.l.store.GetChannel(channelID, channelType)
	if err != nil {
		return nil, err
	}
	if channelInfo != nil || cm.l.opts.CreateIfChannelNotExist {

		if channelInfo == nil {
			channelInfo = NewChannelInfo(channelID, channelType)
		}
		channel := NewChannel(channelInfo, cm.l)
		err := channel.LoadData()
		if err != nil {
			return nil, err
		}
		cm.channelMap.Store(key, channel)
		return channel, nil
	}
	return nil, nil
}

// CreateOrUpdatePersonChannel 创建或更新个人频道
func (cm *ChannelManager) CreateOrUpdatePersonChannel(uid string) error {
	exist, err := cm.l.store.ExistChannel(uid, ChannelTypePerson)
	if err != nil {
		return errors.Wrap(err, "查询是否存在频道信息失败！")
	}
	if !exist {
		err = cm.l.store.AddOrUpdateChannel(&ChannelInfo{
			ChannelID:   uid,
			ChannelType: ChannelTypePerson,
		})
		if err != nil {
			return errors.Wrap(err, "创建个人频道失败！")
		}
	}
	subscribers, err := cm.l.store.GetSubscribers(uid, ChannelTypePerson)
	if err != nil {
		return errors.Wrap(err, "获取频道订阅者失败！")
	}
	if len(subscribers) == 0 || !util.ArrayContains(subscribers, uid) {
		err = cm.l.store.AddSubscribers(uid, ChannelTypePerson, []string{uid})
		if err != nil {
			return errors.Wrap(err, "添加订阅者失败！")
		}
	}
	return nil
}

// CreateOrUpdateChannel 创建或更新频道
func (cm *ChannelManager) CreateOrUpdateChannel(channel *Channel) error {
	err := cm.l.store.AddOrUpdateChannel(channel.ChannelInfo)
	if err != nil {
		return err
	}
	cm.channelMap.Store(fmt.Sprintf("%s-%d", channel.ChannelID, channel.ChannelType), channel)
	return nil
}

// CreateTmpChannel 创建临时频道
func (cm *ChannelManager) CreateTmpChannel(channelID string, channelType uint8, subscribers []string) error {
	channel := NewChannel(NewChannelInfo(channelID, channelType), cm.l)
	if len(subscribers) > 0 {
		for _, subscriber := range subscribers {
			channel.AddSubscriber(subscriber)
		}
	}
	cm.tmpChannelMap.Store(fmt.Sprintf("%s-%d", channelID, channelType), channel)
	return nil
}

// GetPersonChannel 创建临时频道
func (cm *ChannelManager) GetPersonChannel(channelID string, channelType uint8) *Channel {
	v, ok := cm.personChannelMap.Load(fmt.Sprintf("%s-%d", channelID, channelType))
	if ok {
		return v.(*Channel)
	}
	channel := NewChannel(NewChannelInfo(channelID, channelType), cm.l)
	subscribers := strings.Split(channelID, "@")
	for _, subscriber := range subscribers {
		channel.AddSubscriber(subscriber)
	}

	cm.personChannelMap.Store(fmt.Sprintf("%s-%d", channelID, channelType), channel)
	return channel
}

// GetTmpChannel 获取临时频道
func (cm *ChannelManager) GetTmpChannel(channelID string, channelType uint8) (*Channel, error) {
	v, ok := cm.tmpChannelMap.Load(fmt.Sprintf("%s-%d", channelID, channelType))
	if ok {
		return v.(*Channel), nil
	}
	return nil, nil
}

// DeleteChannel 删除频道
func (cm *ChannelManager) DeleteChannel(channelID string, channelType uint8) error {
	err := cm.l.store.DeleteChannel(channelID, channelType)
	if err != nil {
		return err
	}
	cm.DeleteChannelFromCache(channelID, channelType)
	return nil
}

// DeleteChannelFromCache DeleteChannelFromCache
func (cm *ChannelManager) DeleteChannelFromCache(channelID string, channelType uint8) {
	cm.channelMap.Delete(fmt.Sprintf("%s-%d", channelID, channelType))
}
