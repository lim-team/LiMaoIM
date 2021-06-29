package lmproto

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnackEncodeAndDecode(t *testing.T) {
	packet := &ConnackPacket{
		TimeDiff:   12345,
		ReasonCode: ReasonSuccess,
	}
	codec := New()
	// 编码
	packetBytes, err := codec.EncodePacket(packet, 1)
	assert.NoError(t, err)
	fmt.Println(fmt.Sprintf("%x", packetBytes))
	// 解码
	resultPacket, _, err := codec.DecodePacket(packetBytes, 1)
	assert.NoError(t, err)
	resultConnackPacket, ok := resultPacket.(*ConnackPacket)
	assert.Equal(t, true, ok)

	// 正确与否比较
	assert.Equal(t, packet.TimeDiff, resultConnackPacket.TimeDiff)
	assert.Equal(t, packet.ReasonCode, resultConnackPacket.ReasonCode)
}
