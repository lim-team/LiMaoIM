package lmproto

import (
	"fmt"

	"github.com/pkg/errors"
)

// ConnackPacket 连接回执包
type ConnackPacket struct {
	Framer
	ServerKey  string     // 通过客户端的RSA公钥加密的服务端DH公钥
	Salt       string     // salt
	TimeDiff   int64      // 客户端时间与服务器的差值，单位毫秒。
	ReasonCode ReasonCode // 原因码
}

// GetPacketType 获取包类型
func (c ConnackPacket) GetPacketType() PacketType {
	return CONNACK
}
func (c ConnackPacket) String() string {
	return fmt.Sprintf("TimeDiff: %d ReasonCode:%s", c.TimeDiff, c.ReasonCode.String())
}

func encodeConnack(frame Frame, version uint8) ([]byte, error) {
	connack := frame.(*ConnackPacket)
	enc := NewEncoder()
	if version > 2 {
		enc.WriteString(connack.ServerKey)
		enc.WriteString(connack.Salt)
	}
	enc.WriteInt64(connack.TimeDiff)
	enc.WriteByte(connack.ReasonCode.Byte())
	return enc.Bytes(), nil
}

func decodeConnack(frame Frame, data []byte, version uint8) (Frame, error) {
	dec := NewDecoder(data)
	connackPacket := &ConnackPacket{}
	connackPacket.Framer = frame.(Framer)

	var err error

	if version > 2 {
		if connackPacket.ServerKey, err = dec.String(); err != nil {
			return nil, errors.Wrap(err, "解码ServerKey失败！")
		}
		if connackPacket.Salt, err = dec.String(); err != nil {
			return nil, errors.Wrap(err, "解码Salt失败！")
		}
	}

	if connackPacket.TimeDiff, err = dec.Int64(); err != nil {
		return nil, errors.Wrap(err, "解码TimeDiff失败！")
	}
	var reasonCode uint8
	if reasonCode, err = dec.Uint8(); err != nil {
		return nil, errors.Wrap(err, "解码ReasonCode失败！")
	}
	connackPacket.ReasonCode = ReasonCode(reasonCode)
	return connackPacket, nil
}
