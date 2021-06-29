package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPoolGetSlot(t *testing.T) {
	dir, err := ioutil.TempDir("", "commitlog-index")
	assert.NoError(t, err)
	p := NewPool(dir, 1024*1024)
	defer func() {
		p.Close()
		os.Remove(dir)
	}()

	for i := 0; i < 10; i++ {
		m := &Message{
			MessageID:  int64(i),
			MessageSeq: uint32(i + 1000),
			FromUID:    fmt.Sprintf("test-%d", i),
		}
		_, err := p.GetSlot(uint(i)).GetTopic(fmt.Sprintf("topic-%d", i)).AppendLog(m)
		assert.NoError(t, err)
	}
}
