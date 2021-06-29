package lim

import (
	"fmt"
	"sort"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/lim-team/LiMaoIM/internal/db"
	"github.com/lim-team/LiMaoIM/pkg/keylock"
	"github.com/lim-team/LiMaoIM/pkg/limlog"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"go.uber.org/zap"
)

// ConversationManager ConversationManager
type ConversationManager struct {
	channelLock *keylock.KeyLock
	l           *LiMao
	limlog.Log
	queue                          *Queue
	userConversationMapBuckets     []map[string]*lru.Cache
	userConversationMapBucketLocks []sync.RWMutex
	bucketNum                      int
	needSaveConversationMap        map[string]bool
	needSaveConversationMapLock    sync.RWMutex
	stopChan                       chan struct{} //停止信号
	calcChan                       chan interface{}
	needSaveChan                   chan string
}

// NewConversationManager NewConversationManager
func NewConversationManager(l *LiMao) *ConversationManager {
	cm := &ConversationManager{
		l:                       l,
		bucketNum:               10,
		Log:                     limlog.NewLIMLog("ConversationManager"),
		channelLock:             keylock.NewKeyLock(),
		needSaveConversationMap: map[string]bool{},
		stopChan:                make(chan struct{}),
		calcChan:                make(chan interface{}),
		needSaveChan:            make(chan string),
		queue:                   NewQueue(),
	}
	cm.userConversationMapBuckets = make([]map[string]*lru.Cache, cm.bucketNum)
	cm.userConversationMapBucketLocks = make([]sync.RWMutex, cm.bucketNum)

	return cm
}

// Start Start
func (cm *ConversationManager) Start() {
	cm.channelLock.StartCleanLoop()
	go cm.saveloop()
	go cm.calcLoop()
}

// Stop Stop
func (cm *ConversationManager) Stop() {
	close(cm.stopChan)
	cm.channelLock.StopCleanLoop()
	// Wait for the queue to complete
	cm.queue.Wait()

	cm.saveConversations()

}

// 保存最近会话
func (cm *ConversationManager) calcLoop() {
	for {
		messageMapObj := cm.queue.Pop()
		if messageMapObj == nil {
			continue
		}
		messageMap := messageMapObj.(map[string]interface{})
		message := messageMap["message"].(*Message)
		subscribers := messageMap["subscribers"].([]string)

		for _, subscriber := range subscribers {
			cm.calConversation(message, subscriber)
		}
	}
}

func (cm *ConversationManager) saveloop() {
	ticker := time.NewTicker(cm.l.opts.ConversationSyncInterval.Duration)

	needSync := false
	noSaveCount := 0
	for {
		if noSaveCount >= cm.l.opts.ConversationSyncOnce {
			needSync = true
		}
		if needSync {
			noSaveCount = 0
			cm.saveConversations()
		}
		select {
		case uid := <-cm.needSaveChan:
			cm.needSaveConversationMapLock.Lock()
			if !cm.needSaveConversationMap[uid] {
				cm.needSaveConversationMap[uid] = true
				noSaveCount++
			}
			cm.needSaveConversationMapLock.Unlock()

		case <-ticker.C:
			if noSaveCount > 0 {
				needSync = true
			}
		case <-cm.stopChan:
			return
		}
	}
}

// PushMessage PushMessage
func (cm *ConversationManager) PushMessage(message *Message, subscribers []string) {
	if len(subscribers) == 0 || message == nil {
		return
	}
	if !message.SyncOnce && !message.NoPersist {
		cm.queue.Push(map[string]interface{}{
			"message":     message,
			"subscribers": subscribers,
		})
	}
}

// SetConversationUnread set unread data from conversation
func (cm *ConversationManager) SetConversationUnread(uid string, channelID string, channelType uint8, unread int) error {
	conversationCache := cm.getUserConversationCache(uid)
	for _, key := range conversationCache.Keys() {
		channelKey := cm.getChannelKey(channelID, channelType)
		if channelKey == key {
			conversationObj, ok := conversationCache.Get(key)
			if ok {
				conversationObj.(*db.Conversation).UnreadCount = unread
				cm.setNeedSave(uid)
			}
			break
		}
	}
	return nil
}

// DeleteConversation 删除最近会话
func (cm *ConversationManager) DeleteConversation(uids []string, channelID string, channelType uint8) error {
	if len(uids) == 0 {
		return nil
	}
	for _, uid := range uids {
		conversationCache := cm.getUserConversationCache(uid)
		keys := conversationCache.Keys()
		for _, key := range keys {
			channelKey := cm.getChannelKey(channelID, channelType)
			if channelKey == key {
				conversationCache.Remove(key)
				break
			}
		}
	}
	return nil
}

func (cm *ConversationManager) getUserAllConversationMapFromStore(uid string) ([]*db.Conversation, error) {
	conversations, err := cm.l.store.GetConversations(uid)
	if err != nil {
		cm.Error("Failed to get the list of recent conversations", zap.String("uid", uid), zap.Error(err))
		return nil, err
	}
	return conversations, nil
}

func (cm *ConversationManager) newLRUCache() *lru.Cache {
	c, _ := lru.New(cm.l.opts.ConversationOfUserMaxCount)
	return c
}

// 保存最近会话
func (cm *ConversationManager) saveConversations() {

	cm.needSaveConversationMapLock.RLock()
	needSaveUIDs := make([]string, 0, len(cm.needSaveConversationMap))
	for uid := range cm.needSaveConversationMap {
		needSaveUIDs = append(needSaveUIDs, uid)
	}
	cm.needSaveConversationMapLock.RUnlock()

	if len(needSaveUIDs) > 0 {
		cm.Debug("Save conversation", zap.Int("count", len(needSaveUIDs)))
		for _, uid := range needSaveUIDs {
			conversationCache := cm.getUserConversationCache(uid)
			conversations := make([]*db.Conversation, 0, len(conversationCache.Keys()))
			for _, key := range conversationCache.Keys() {
				conversationObj, ok := conversationCache.Get(key)
				if ok {
					conversations = append(conversations, conversationObj.(*db.Conversation))
				}

			}
			err := cm.l.store.AddOrUpdateConversations(uid, conversations)
			if err != nil {
				cm.Warn("Failed to store conversation data", zap.Error(err))
			} else {
				cm.needSaveConversationMapLock.Lock()
				delete(cm.needSaveConversationMap, uid)
				cm.needSaveConversationMapLock.Unlock()
			}
		}
	}

}

func (cm *ConversationManager) getUserConversationCache(uid string) *lru.Cache {
	pos := int(util.HashCrc32(uid) % uint32(cm.bucketNum))
	cm.userConversationMapBucketLocks[pos].RLock()
	userConversationMap := cm.userConversationMapBuckets[pos]
	if userConversationMap == nil {
		userConversationMap = make(map[string]*lru.Cache)
		cm.userConversationMapBuckets[pos] = userConversationMap
	}

	cm.userConversationMapBucketLocks[pos].RUnlock()
	cm.channelLock.Lock(uid)
	cache := userConversationMap[uid]
	if cache == nil {
		cache = cm.newLRUCache()
		userConversationMap[uid] = cache
	}
	cm.channelLock.Unlock(uid)

	return cache
}

func (cm *ConversationManager) calConversation(message *Message, subscriber string) {
	conversationCache := cm.getUserConversationCache(subscriber)

	if conversationCache.Len() == 0 {
		var err error
		conversations, err := cm.getUserAllConversationMapFromStore(subscriber)
		if err != nil {
			cm.Warn("Failed to get the conversation from the database", zap.Error(err))
			return
		}
		for _, conversation := range conversations {
			channelKey := cm.getChannelKey(conversation.ChannelID, conversation.ChannelType)
			conversationCache.Add(channelKey, conversation)

		}
	}

	channelID := message.ChannelID
	if message.ChannelType == ChannelTypePerson && message.ChannelID == subscriber { // If it is a personal channel and the channel ID is equal to the subscriber, you need to swap fromUID and channelID
		channelID = message.FromUID
	}
	channelKey := cm.getChannelKey(channelID, message.ChannelType)

	cm.channelLock.Lock(channelKey)
	conversationCache.Get(channelKey)
	var conversation *db.Conversation
	conversationObj, ok := conversationCache.Get(channelKey)
	if ok {
		conversation = conversationObj.(*db.Conversation)
	}
	cm.channelLock.Unlock(channelKey)
	unreadCount := 0
	if message.RedDot {
		unreadCount = 1
	}
	if conversation == nil {
		conversation = &db.Conversation{
			UID:             subscriber,
			ChannelID:       channelID,
			ChannelType:     message.ChannelType,
			UnreadCount:     unreadCount,
			Timestamp:       int64(message.Timestamp),
			LastMsgSeq:      message.MessageSeq,
			LastClientMsgNo: message.ClientMsgNo,
			LastMsgID:       message.MessageID,
			Version:         time.Now().UnixNano() / 1e6,
		}
		cm.setNeedSave(subscriber)

	} else {
		if message.RedDot {
			conversation.UnreadCount++
			cm.setNeedSave(subscriber)
		}
	}
	cm.channelLock.Lock(channelKey)
	conversationCache.Add(channelKey, conversation)
	cm.channelLock.Unlock(channelKey)

}

// GetConversations GetConversations
func (cm *ConversationManager) GetConversations(uid string, version int64) []*db.Conversation {
	conversationCache := cm.getUserConversationCache(uid)

	conversationSlice := conversationSlice{}
	fmt.Println("conversationCache.Len()--->", conversationCache.Len())
	if conversationCache.Len() == 0 {
		var err error
		conversations, err := cm.getUserAllConversationMapFromStore(uid)
		if err != nil {
			cm.Warn("Failed to get the conversation from the database", zap.Error(err))
			return nil
		}
		for _, conversation := range conversations {
			channelKey := cm.getChannelKey(conversation.ChannelID, conversation.ChannelType)
			conversationCache.Add(channelKey, conversation)

		}
	}
	for _, key := range conversationCache.Keys() {
		conversationObj, ok := conversationCache.Get(key)
		if ok {
			conversation := conversationObj.(*db.Conversation)
			if version <= 0 || conversation.Version > version {
				conversationSlice = append(conversationSlice, conversation)
			}
		}
	}
	sort.Sort(conversationSlice)
	return conversationSlice
}

func (cm *ConversationManager) setNeedSave(uid string) {
	cm.needSaveChan <- uid
}

func (cm *ConversationManager) getChannelKey(channelID string, channelType uint8) string {
	return fmt.Sprintf("%s-%d", channelID, channelType)
}

type conversationSlice []*db.Conversation

func (s conversationSlice) Len() int { return len(s) }

func (s conversationSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s conversationSlice) Less(i, j int) bool {
	return s[i].Timestamp > s[j].Timestamp
}
