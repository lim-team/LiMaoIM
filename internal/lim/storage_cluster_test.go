package lim

import (
	"testing"

	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/stretchr/testify/assert"
)

func TestUpdateUserToken(t *testing.T) {
	opts := NewTestOptions()
	opts.IsCluster = true
	l := NewTestLiMao(opts)
	l.DoCommand = func(cmd *CMD) error {
		assert.Equal(t, CMDUpdateUserToken, cmd.Type)
		uid, flag, level, token, err := cmd.DecodeUserToken()
		assert.NoError(t, err)
		err = l.store.GetFileStorage().UpdateUserToken(uid, flag, level, token)
		assert.NoError(t, err)
		return nil
	}
	err := l.store.UpdateUserToken("test", lmproto.APP, lmproto.DeviceLevelMaster, "token123")
	assert.NoError(t, err)

	token, level, err := l.store.GetUserToken("test", lmproto.APP)
	assert.NoError(t, err)

	assert.Equal(t, "token123", token)
	assert.Equal(t, lmproto.DeviceLevelMaster, level)
}
