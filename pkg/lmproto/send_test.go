package lmproto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendEncodeAndDecode(t *testing.T) {

	packet := &SendPacket{
		Framer: Framer{
			RedDot: true,
		},
		ClientSeq:   2,
		ChannelID:   "3434",
		ChannelType: 2,
		Payload:     []byte("dsdsdsd"),
	}
	packet.RedDot = true

	codec := New()
	// 编码
	packetBytes, err := codec.EncodePacket(packet, 1)
	assert.NoError(t, err)

	// 解码
	resultPacket, _, err := codec.DecodePacket(packetBytes, 1)
	assert.NoError(t, err)
	resultSendPacket, ok := resultPacket.(*SendPacket)
	assert.Equal(t, true, ok)

	// 比较
	assert.Equal(t, packet.ClientSeq, resultSendPacket.ClientSeq)
	assert.Equal(t, packet.ChannelID, resultSendPacket.ChannelID)
	assert.Equal(t, packet.ChannelType, resultSendPacket.ChannelType)
	assert.Equal(t, packet.RedDot, resultSendPacket.RedDot)
	assert.Equal(t, packet.Payload, resultSendPacket.Payload)
}
