package lmproto

import (
	"fmt"

	"github.com/pkg/errors"
)

// RecvackPacket 对收取包回执
type RecvackPacket struct {
	Framer
	MessageID  int64  // 服务端的消息ID(全局唯一)
	MessageSeq uint32 // 消息序列号
}

// GetPacketType 包类型
func (s *RecvackPacket) GetPacketType() PacketType {
	return RECVACK
}

func (s *RecvackPacket) String() string {
	return fmt.Sprintf("MessageId:%d MessageSeq:%d", s.MessageID, s.MessageSeq)
}

func decodeRecvack(frame Frame, data []byte, version uint8) (Frame, error) {
	dec := NewDecoder(data)
	recvackPacket := &RecvackPacket{}
	recvackPacket.Framer = frame.(Framer)
	var err error
	// 消息唯一ID
	if recvackPacket.MessageID, err = dec.Int64(); err != nil {
		return nil, errors.Wrap(err, "解码MessageId失败！")
	}
	// 消息唯序列号
	if recvackPacket.MessageSeq, err = dec.Uint32(); err != nil {
		return nil, errors.Wrap(err, "解码MessageSeq失败！")
	}
	return recvackPacket, err
}

func encodeRecvack(frame Frame, version uint8) ([]byte, error) {
	recvackPacket := frame.(*RecvackPacket)
	enc := NewEncoder()
	enc.WriteInt64(recvackPacket.MessageID)
	enc.WriteUint32(recvackPacket.MessageSeq)
	return enc.Bytes(), nil
}
