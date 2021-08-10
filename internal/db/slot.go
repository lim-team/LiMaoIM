package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/lim-team/LiMaoIM/pkg/keylock"
	"github.com/tangtaoit/limnet/pkg/limlog"
)

// Slot Slot
type Slot struct {
	slot            uint32
	segmentMaxBytes int64
	slotDir         string
	lruCache        *lru.Cache
	segmentLock     sync.Mutex
	limlog.Log
	lock *keylock.KeyLock
}

// NewSlot NewSlot
func NewSlot(dataDir string, slot uint32, segmentMaxBytes int64) *Slot {
	cache, err := lru.NewWithEvict(1000, func(key, value interface{}) {
		topic := value.(*Topic)
		topic.Close()
	})
	if err != nil {
		panic(err)
	}
	s := &Slot{
		slot:            slot,
		lruCache:        cache,
		lock:            keylock.NewKeyLock(),
		segmentMaxBytes: segmentMaxBytes,
		Log:             limlog.NewLIMLog(fmt.Sprintf("Slot-%d", slot)),
		slotDir:         filepath.Join(dataDir, "slots", fmt.Sprintf("%d", slot)),
	}
	_, err = os.Stat(s.slotDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(s.slotDir, FileDefaultMode)
		if err != nil {
			s.Error("mkdir slot dir fail")
			panic(err)
		}
	}

	return s
}

// GetTopic GetTopic
func (s *Slot) GetTopic(topic string) *Topic {
	s.lock.Lock(topic)
	defer s.lock.Unlock(topic)
	v, ok := s.lruCache.Get(topic)
	if ok {
		return v.(*Topic)
	}
	t := NewTopic(s.slotDir, topic, s.segmentMaxBytes)
	s.lruCache.Add(topic, t)
	return t
}

// Close Close
func (s *Slot) Close() error {

	keys := s.lruCache.Keys()
	for _, key := range keys {
		s.lruCache.Remove(key) // Trigger onEvicted method
	}
	return nil
}

// Sync Sync
func (s *Slot) Sync() error {
	keys := s.lruCache.Keys()
	for _, key := range keys {
		v, ok := s.lruCache.Get(key) // Trigger onEvicted method
		if ok {
			err := v.(*Topic).Sync()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
