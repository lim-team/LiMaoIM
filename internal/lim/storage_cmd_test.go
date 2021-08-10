package lim

import (
	"testing"

	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/stretchr/testify/assert"
)

func TestEncodeUserToken(t *testing.T) {
	data := NewCMD(CMDUpdateUserToken).EncodeUserToken("test", lmproto.APP, lmproto.DeviceLevelMaster, "token123").Encode()

	cmd := &CMD{}
	UnmarshalCMD(data, cmd)

	uid, deviceFlag, deviceLevel, token, err := cmd.DecodeUserToken()
	assert.NoError(t, err)

	assert.Equal(t, "test", uid)
	assert.Equal(t, lmproto.APP, deviceFlag)
	assert.Equal(t, lmproto.DeviceLevelMaster, deviceLevel)
	assert.Equal(t, "token123", token)
}
