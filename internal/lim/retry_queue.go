package lim

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// RetryQueue 重试队列
type RetryQueue struct {
	inFlightPQ       inFlightPqueue
	inFlightMessages map[string]*Message
	inFlightMutex    sync.Mutex
	l                *LiMao
}

// NewRetryQueue NewRetryQueue
func NewRetryQueue(l *LiMao) *RetryQueue {

	return &RetryQueue{
		inFlightPQ:       newInFlightPqueue(1024),
		inFlightMessages: make(map[string]*Message),
		l:                l,
	}
}

func (r *RetryQueue) startInFlightTimeout(msg *Message) {
	now := time.Now()
	msg.pri = now.Add(r.l.opts.MsgTimeout.Duration).UnixNano()
	r.pushInFlightMessage(msg)
	r.addToInFlightPQ(msg)
}

func (r *RetryQueue) addToInFlightPQ(msg *Message) {
	r.inFlightMutex.Lock()
	defer r.inFlightMutex.Unlock()
	r.inFlightPQ.Push(msg)

}
func (r *RetryQueue) pushInFlightMessage(msg *Message) {
	r.inFlightMutex.Lock()
	defer r.inFlightMutex.Unlock()
	key := fmt.Sprintf("%d_%d", msg.MessageID, msg.toClientID)
	_, ok := r.inFlightMessages[key]
	if ok {
		return
	}
	r.inFlightMessages[key] = msg

}

func (r *RetryQueue) popInFlightMessage(clientID int64, messageID int64) (*Message, error) {
	r.inFlightMutex.Lock()
	defer r.inFlightMutex.Unlock()
	key := fmt.Sprintf("%d_%d", messageID, clientID)
	msg, ok := r.inFlightMessages[key]
	if !ok {
		return nil, errors.New("ID not in flight")
	}
	delete(r.inFlightMessages, key)
	return msg, nil
}
func (r *RetryQueue) finishMessage(clientID int64, messageID int64) error {
	msg, err := r.popInFlightMessage(clientID, messageID)
	if err != nil {
		return err
	}
	r.removeFromInFlightPQ(msg)

	return nil
}
func (r *RetryQueue) removeFromInFlightPQ(msg *Message) {
	r.inFlightMutex.Lock()
	if msg.index == -1 {
		// this item has already been popped off the pqueue
		r.inFlightMutex.Unlock()
		return
	}
	r.inFlightPQ.Remove(msg.index)
	r.inFlightMutex.Unlock()
}

func (r *RetryQueue) processInFlightQueue(t int64) {
	for {
		r.inFlightMutex.Lock()
		msg, _ := r.inFlightPQ.PeekAndShift(t)
		r.inFlightMutex.Unlock()
		if msg == nil {
			break
		}
		_, err := r.popInFlightMessage(msg.toClientID, msg.MessageID)
		if err != nil {
			break
		}
		r.l.startDeliveryMsg(msg, nil, msg.ToUID)
	}
}

// Start 开始运行重试
func (r *RetryQueue) Start() {
	r.l.Schedule(r.l.opts.TimeoutScanInterval.Duration, func() {
		now := time.Now().UnixNano()
		r.processInFlightQueue(now)
	})
}
