package lmproxyproto

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestDecodeAndEncodeWithReader(t *testing.T) {
	p := NewProtocol()
	req := &ConnectReq{
		NodeID: 10,
	}
	data, _ := proto.Marshal(req)
	reqCMD := &CMD{
		ID:    232323,
		Cmd:   CMDTypeConnect,
		Param: data,
	}
	encdoeData, err := p.Encode(reqCMD)
	assert.NoError(t, err)

	resultCMD, err := p.DecodeWithReader(bytes.NewBuffer(encdoeData))
	assert.NoError(t, err)

	assert.Equal(t, resultCMD.Cmd, CMDTypeConnect)
	resultConnectReq := &ConnectReq{}
	err = proto.Unmarshal(resultCMD.Param, resultConnectReq)
	assert.NoError(t, err)
	assert.Equal(t, req.NodeID, resultConnectReq.NodeID)
	assert.Equal(t, reqCMD.ID, resultCMD.ID)

}
