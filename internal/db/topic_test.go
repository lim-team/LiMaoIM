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

func TestReadLastLogs(t *testing.T) {
	dir, err := ioutil.TempDir("", "commitlog-index")
	assert.NoError(t, err)

	var maxBytesPerLogFile int64 = 1024
	tc := NewTopic(dir, "test", maxBytesPerLogFile)

	defer func() {
		tc.Close()
		os.Remove(dir)
	}()
	var totalBytes int64 = 0
	for i := 0; i < 1000; i++ {
		seq, _ := tc.NextOffset()
		m := &Message{
			MessageID:   int64(i),
			MessageSeq:  seq,
			ChannelID:   fmt.Sprintf("test%d", i),
			ChannelType: 2,
			FromUID:     fmt.Sprintf("test-%d", i),
			Payload:     []byte("this is test"),
		}
		n, err := tc.AppendLog(m)
		assert.NoError(t, err)
		totalBytes += int64(n)
	}
	var messages = make([]*Message, 0)
	err = tc.ReadLastLogs(1000, func(data []byte) error {
		m := &Message{}
		err := UnmarshalMessage(data, m)
		if err != nil {
			return err
		}
		messages = append(messages, m)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1000, len(messages))
	assert.Equal(t, 1000, int(messages[999].MessageSeq))

}

// func TestReadLastLogs2(t *testing.T) {
// 	dir, err := ioutil.TempDir("", "commitlog-index")
// 	assert.NoError(t, err)

// 	var maxBytesPerLogFile int64 = 1024
// 	tc := NewTopic(dir, "test", maxBytesPerLogFile)

// 	defer func() {
// 		tc.Close()
// 		os.Remove(dir)
// 	}()

// 	for i := 0; i < 41; i++ {
// 		seq, _ := tc.NextOffset()
// 		m := &Message{
// 			MessageID:   int64(i),
// 			MessageSeq:  seq,
// 			ChannelID:   fmt.Sprintf("test%d", i),
// 			ChannelType: 2,
// 			FromUID:     fmt.Sprintf("test-%d", i),
// 			Payload:     []byte(fmt.Sprintf("%d", i+1)),
// 		}
// 		_, err := tc.AppendLog(m)
// 		assert.NoError(t, err)
// 	}
// 	var messages = make([]*Message, 0)
// 	err = tc.ReadLastLogs(20, func(data []byte) error {
// 		m := &Message{}
// 		err := UnmarshalMessage(data, m)
// 		if err != nil {
// 			return err
// 		}
// 		messages = append(messages, m)
// 		return nil
// 	})
// 	assert.NoError(t, err)
// 	assert.Equal(t, 20, len(messages))
// }
