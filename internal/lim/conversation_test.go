package lim

import (
	"testing"
	"time"

	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/stretchr/testify/assert"
)

func TestGetConversations(t *testing.T) {
	opts := NewTestOptions()
	opts.ConversationSyncOnce = 0
	l := NewTestLiMao(opts)
	cm := NewConversationManager(l)
	cm.Start()

	defer cm.Stop()
	m := &Message{
		RecvPacket: lmproto.RecvPacket{
			Framer: lmproto.Framer{
				RedDot: true,
			},
			MessageID:   123,
			ChannelID:   "group1",
			ChannelType: 2,
			FromUID:     "test",
			Timestamp:   int32(time.Now().Unix()),
			Payload:     []byte("hello"),
		},
	}
	cm.PushMessage(m, []string{"test"})

	m = &Message{
		RecvPacket: lmproto.RecvPacket{
			Framer: lmproto.Framer{
				RedDot: true,
			},
			MessageID:   123,
			ChannelID:   "group2",
			ChannelType: 2,
			FromUID:     "test",
			Timestamp:   int32(time.Now().Unix()),
			Payload:     []byte("hello"),
		},
	}
	cm.PushMessage(m, []string{"test"})

	time.Sleep(time.Millisecond * 100) // wait calc conversation

	cm.l.store.Close()

	conversations := cm.GetConversations("test", 0)
	assert.Equal(t, 2, len(conversations))

}
