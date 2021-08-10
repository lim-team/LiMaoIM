package lim

import (
	"testing"

	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/stretchr/testify/assert"
)

func TestUpdateAndGetUserToken(t *testing.T) {
	opts := NewTestOptions()
	l := NewTestLiMao(opts)

	err := l.store.UpdateUserToken("test", lmproto.APP, lmproto.DeviceLevelMaster, "testtoken")
	assert.NoError(t, err)

	token, level, err := l.store.GetUserToken("test", lmproto.APP)
	assert.NoError(t, err)

	assert.Equal(t, "testtoken", token)
	assert.Equal(t, lmproto.DeviceLevelMaster, level)
}
