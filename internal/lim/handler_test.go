package lim

import (
	"fmt"
	"sync"
	"testing"

	"github.com/lim-team/LiMaoIM/pkg/client"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/stretchr/testify/assert"
)

func TestHandleConnect(t *testing.T) {
	l := NewTestLiMao()
	l.Start()
	defer l.Stop()

	MustConnectLiMao(l, "test")
}

func TestSendMessage(t *testing.T) {
	l := NewTestLiMao()
	l.Start()
	defer l.Stop()

	c1 := MustConnectLiMao(l, "test1")
	defer c1.Disconnect()
	c2 := MustConnectLiMao(l, "test2")
	defer c2.Disconnect()

	err := c1.SendMessage(client.NewChannel("test2", 1), []byte(fmt.Sprintf("hello")))
	assert.NoError(t, err)

	var wait sync.WaitGroup

	wait.Add(1)
	c2.SetOnRecv(func(recv *lmproto.RecvPacket) error {
		assert.Equal(t, "hello", string(recv.Payload))
		wait.Done()
		return nil
	})
	wait.Wait()
}

func BenchmarkSendMessage(b *testing.B) {
	b.StopTimer()
	l := NewTestLiMao()
	err := l.Start()
	assert.NoError(b, err)
	defer func() {
		err := l.Stop()
		assert.NoError(b, err)

	}()
	c1 := MustConnectLiMao(l, "test1")
	defer c1.Disconnect()
	b.StartTimer()

	var wg sync.WaitGroup
	c1.SetOnSendack(func(sendack *lmproto.SendackPacket) {
		wg.Done()
	})
	for i := 0; i <= b.N; i++ {
		wg.Add(1)
		err := c1.SendMessage(client.NewChannel("test2", 1), []byte(fmt.Sprintf("hello-%d", i)))
		assert.NoError(b, err)
	}

	wg.Wait()
}
