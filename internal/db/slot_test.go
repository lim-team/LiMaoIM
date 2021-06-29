package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlotGetTopic(t *testing.T) {
	dir, err := ioutil.TempDir("", "commitlog-index")
	assert.NoError(t, err)
	st := NewSlot(dir, 1, 1024*1024)
	defer func() {
		st.Close()
		os.Remove(dir)
	}()

	w := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		w.Add(1)
		go func(i int) {
			m := &Message{
				MessageID:   int64(i),
				MessageSeq:  uint32(i),
				ChannelID:   fmt.Sprintf("test%d", i),
				ChannelType: 1,
				FromUID:     fmt.Sprintf("test-%d", i),
				Payload:     []byte("this is test"),
			}
			st.GetTopic(fmt.Sprintf("testTopic-%d", i)).AppendLog(m)
			assert.NoError(t, err)
			w.Done()
		}(i)
	}

	w.Wait()

}
