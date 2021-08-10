package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/bbolt"
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

func TestBackupSlots(t *testing.T) {
	dir, err := ioutil.TempDir("", "commitlog-index")
	assert.NoError(t, err)
	fmt.Println("dir-->", dir)
	fd := NewFileDB(dir, 1024*1024, 256)
	defer func() {
		fd.Close() // close很慢
		os.Remove(dir)
	}()

	for i := 0; i < 100; i++ {
		fd.db.Update(func(t *bbolt.Tx) error {
			b, _ := fd.getSlotBucket(uint32(i), t)
			b.Put([]byte(fmt.Sprintf("key-test-%d", i)), []byte(fmt.Sprintf("value-test-%d", i)))
			return nil
		})
	}

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
	slotMap := util.NewSlotBitMap(fd.slotCount)
	slotMap.SetSlotForRange(0, uint32(fd.slotCount)-1, true)
	f, err := os.OpenFile(path.Join(dir, "test.backup"), os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	err = fd.BackupSlots(slotMap.GetBits(), f)
	assert.NoError(t, err)
}

func TestRecoverSlotBackup(t *testing.T) {
	dir, err := ioutil.TempDir("", "commitlog-index")
	assert.NoError(t, err)
	fmt.Println(dir)
	fd := NewFileDB(dir, 1024*100, 256)
	defer func() {
		fd.Close() // close很慢
		os.Remove(dir)
	}()

	for i := 0; i < 100; i++ {
		fd.db.Update(func(t *bbolt.Tx) error {
			b, _ := fd.getSlotBucket(uint32(i), t)
			b.Put([]byte(fmt.Sprintf("key-test-%d", i)), []byte(fmt.Sprintf("value-test-%d", i)))
			return nil
		})
	}

	var size int = 0
	for i := 0; i < 10000; i++ {
		m := &Message{
			MessageID:   int64(i),
			MessageSeq:  uint32(i + 1),
			ChannelID:   fmt.Sprintf("test%d", i%2),
			ChannelType: 1,
			AppliIndex:  uint64(i + 1),
			FromUID:     fmt.Sprintf("test%d", i),
			Payload:     []byte("this is test"),
		}
		n, err := fd.AppendMessage(m)
		assert.NoError(t, err)
		size += n
	}
	slotMap := util.NewSlotBitMap(fd.slotCount)
	slotMap.SetSlot(62, true)
	backupDir := path.Join(dir, "test.backup")
	f, err := os.OpenFile(backupDir, os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	err = fd.BackupSlots(slotMap.GetBits(), f)
	assert.NoError(t, err)
	f.Close()

	err = os.RemoveAll(path.Join(fd.dataDir, "slots"))
	assert.NoError(t, err)
	os.Remove(path.Join(fd.dataDir, "limaoim.db"))
	assert.NoError(t, err)

	newfd := NewFileDB(dir, 1024*100, 256)
	backupFile, err := os.OpenFile(backupDir, os.O_RDONLY, 0644)
	assert.NoError(t, err)

	err = newfd.RecoverSlotBackup(backupFile)
	assert.NoError(t, err)

	newfd.Close()

}
