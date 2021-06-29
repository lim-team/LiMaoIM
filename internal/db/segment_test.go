package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSegmentAppend(t *testing.T) {
	dir, err := ioutil.TempDir("", "commitlog-index")
	assert.NoError(t, err)
	sg := NewSegment(dir, 0)

	defer func() {
		sg.Close()
		os.Remove(dir)
	}()

	for i := 0; i < 100000; i++ {
		m := &Message{
			MessageID:  int64(i),
			MessageSeq: uint32(i + 1000),
			FromUID:    fmt.Sprintf("dddzz-%d", i),
		}
		_, err := sg.AppendLog(m)
		assert.NoError(t, err)
	}

	var messageSeq uint32 = 1230
	resultMessage := &Message{}
	err = sg.ReadAt(int64(messageSeq), resultMessage)
	assert.NoError(t, err)
	assert.Equal(t, messageSeq, resultMessage.MessageSeq)
}

func TestSegmentSanityCheck(t *testing.T) {
	dir, err := ioutil.TempDir("", "commitlog-index")
	assert.NoError(t, err)
	sg := NewSegment(dir, 0)

	defer func() {
		sg.Close()
		os.Remove(dir)
	}()

	for i := 0; i < 100000; i++ {
		m := &Message{
			MessageID:  int64(i),
			MessageSeq: uint32(i + 1000),
			FromUID:    fmt.Sprintf("test-%d", i),
		}
		_, err := sg.AppendLog(m)
		assert.NoError(t, err)
	}
	_, err = sg.SanityCheck()
	assert.NoError(t, err)

	sg.logFile.Truncate(sg.position - 8)

	_, err = sg.SanityCheck()
	assert.NoError(t, err)

}

func TestSegmentReadLogs(t *testing.T) {
	dir, err := ioutil.TempDir("", "commitlog-index")
	assert.NoError(t, err)
	sg := NewSegment(dir, 0)

	defer func() {
		sg.Close()
		os.Remove(dir)
	}()
	for i := 0; i < 100000; i++ {
		m := &Message{
			MessageID:  int64(i),
			MessageSeq: uint32(i + 1000),
			FromUID:    fmt.Sprintf("test-%d", i),
		}
		_, err := sg.AppendLog(m)
		assert.NoError(t, err)
	}

	messages := make([]*Message, 0)
	err = sg.ReadLogs(2001, 40, func(data []byte) error {
		m := &Message{}
		err = m.Decode(data)
		if err != nil {
			return err
		}
		messages = append(messages, m)
		return nil
	})
	assert.NoError(t, err)

	assert.Equal(t, 40, len(messages))

}
