package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/stretchr/testify/assert"
)

func TestUpdateUserToken(t *testing.T) {
	dir, err := ioutil.TempDir("", "commitlog-index")
	assert.NoError(t, err)
	fd := NewFileDB("test", 1024*1024, 256)
	defer func() {
		fd.Close()
		os.Remove(dir)
	}()

	uid := "uid1234"
	token := "token12345"
	err = fd.UpdateUserToken(uid, lmproto.APP, lmproto.DeviceLevelMaster, token)
	assert.NoError(t, err)

	acttoken, level, err := fd.GetUserToken(uid, lmproto.APP)
	assert.NoError(t, err)
	assert.Equal(t, token, acttoken)
	assert.Equal(t, lmproto.DeviceLevelMaster, level)

}

func TestGetMetaData(t *testing.T) {
	dir, err := ioutil.TempDir("", "commitlog-index")
	assert.NoError(t, err)
	fd := NewFileDB(dir, 1024*1024, 256)
	defer func() {
		fd.Close()
		os.Remove(dir)
	}()

	_, err = fd.GetMetaData()
	assert.NoError(t, err)
}

func TestSaveSnapshot(t *testing.T) {
	dir, err := ioutil.TempDir("", "commitlog-index")
	assert.NoError(t, err)
	fd := NewFileDB(dir, 1024*1024, 256)
	defer func() {
		fd.Close() // close很慢
		os.Remove(dir)
	}()

	var size int = 0
	for i := 0; i < 100; i++ {
		m := &Message{
			MessageID:   int64(i),
			MessageSeq:  uint32(i + 1),
			ChannelID:   fmt.Sprintf("test%d", i),
			ChannelType: 1,
			AppliIndex:  uint64(i + 1),
			FromUID:     fmt.Sprintf("test%d", i),
			Payload:     []byte("this is test"),
		}
		n, err := fd.AppendMessage(m)
		assert.NoError(t, err)
		size += n
	}
	f, err := os.OpenFile(filepath.Join(dir, "snapshot.log"), os.O_CREATE|os.O_RDWR, 0755)
	assert.NoError(t, err)
	err = fd.SaveSnapshot(NewSnapshot(100000), f)
	assert.NoError(t, err)

	f.Close()

}
