package db

import (

	// mmapgo "github.com/edsrzf/mmap-go"

	"os"

	lru "github.com/hashicorp/golang-lru"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"go.uber.org/zap"
)

// Pool Pool
type Pool struct {
	dataDir  string
	lruCache *lru.Cache
	limlog.Log
	segmentMaxBytes int64
}

// NewPool NewPool
func NewPool(dataDir string, segmentMaxBytes int64) *Pool {

	cache, err := lru.NewWithEvict(100, func(key, value interface{}) {
		value.(*Slot).Close()
	})
	if err != nil {
		panic(err)
	}
	p := &Pool{
		dataDir:         dataDir,
		segmentMaxBytes: segmentMaxBytes,
		lruCache:        cache,
		Log:             limlog.NewLIMLog("Pool"),
	}
	_, err = os.Stat(dataDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dataDir, FileDefaultMode)
		if err != nil {
			p.Error("mkdir data dir fail", zap.Error(err))
			panic(err)
		}
	}

	return p
}

// GetSlot GetSlot
func (p *Pool) GetSlot(slot uint) *Slot {
	v, ok := p.lruCache.Get(slot)
	if ok {
		return v.(*Slot)
	}
	st := NewSlot(p.dataDir, slot, p.segmentMaxBytes)
	p.lruCache.Add(slot, st)
	return st
}

// Sync Sync
func (p *Pool) Sync() error {
	keys := p.lruCache.Keys()
	for _, key := range keys {
		v, ok := p.lruCache.Get(key)
		if ok {
			err := v.(*Slot).Sync()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Close Close
func (p *Pool) Close() error {

	keys := p.lruCache.Keys()
	for _, key := range keys {
		p.lruCache.Remove(key) // Trigger onEvicted method
	}
	return nil
}
