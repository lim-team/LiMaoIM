package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopicAppend(t *testing.T) {
	dir, err := ioutil.TempDir("", "commitlog-index")
	assert.NoError(t, err)

	var maxBytesPerLogFile int64 = 1024 * 1024
	tc := NewTopic(dir, "test", maxBytesPerLogFile)

	defer func() {
		tc.Close()
		os.Remove(dir)
	}()
	var totalBytes int64 = 0
	for i := 0; i < 100000; i++ {
		m := &Message{
			MessageID:   int64(i),
			MessageSeq:  uint32(i),
			ChannelID:   fmt.Sprintf("test%d", i),
			ChannelType: 2,
			FromUID:     fmt.Sprintf("test-%d", i),
			Payload:     []byte("this is test"),
		}
		n, err := tc.AppendLog(m)
		assert.NoError(t, err)
		totalBytes += int64(n)
	}

	expectFileNum := totalBytes / maxBytesPerLogFile
	if totalBytes%maxBytesPerLogFile != 0 {
		expectFileNum++
	}

	assert.Equal(t, int(expectFileNum), len(tc.segments))

}
