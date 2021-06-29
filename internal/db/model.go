package db

import "github.com/lim-team/LiMaoIM/pkg/lmproto"

// ILog ILog
type ILog interface {
	GetAppliIndex() uint64
	Offset() int64
	Encode() ([]byte, error)
	Decode(data []byte) error
}

// Message Message
type Message struct {
	Header      uint8
	AppliIndex  uint64
	Version     uint8
	Setting     uint8
	MessageID   int64  // 服务端的消息ID(全局唯一)
	MessageSeq  uint32 // 消息序列号 （用户唯一，有序递增）
	ClientMsgNo string // 客户端唯一标示
	Timestamp   int32  // 服务器消息时间戳(10位，到秒)
	FromUID     string // 发送者UID
	ChannelID   string // 频道ID
	ChannelType uint8  // 频道类型
	Payload     []byte // 消息内容
}

// Offset Offset
func (m *Message) Offset() int64 {
	return int64(m.MessageSeq)
}

// Encode Encode
func (m *Message) Encode() ([]byte, error) {
	return MarshalMessage(m)
}

// Decode Decode
func (m *Message) Decode(data []byte) error {
	return UnmarshalMessage(data, m)
}

// GetAppliIndex GetAppliIndex
func (m *Message) GetAppliIndex() uint64 {
	return m.AppliIndex
}

// MarshalMessage MarshalMessage
func MarshalMessage(m *Message) ([]byte, error) {
	enc := lmproto.NewEncoder()
	enc.WriteByte(m.Header)
	enc.WriteUint8(m.Version)
	enc.WriteByte(m.Setting)
	enc.WriteInt64(m.MessageID)
	enc.WriteUint32(m.MessageSeq)
	enc.WriteString(m.ClientMsgNo)
	enc.WriteInt32(m.Timestamp)
	enc.WriteString(m.FromUID)
	enc.WriteString(m.ChannelID)
	enc.WriteUint8(m.ChannelType)
	enc.WriteBytes(m.Payload)
	return enc.Bytes(), nil
}

// UnmarshalMessage UnmarshalMessage
func UnmarshalMessage(data []byte, m *Message) error {
	dec := lmproto.NewDecoder(data)
	var err error
	if m.Header, err = dec.Uint8(); err != nil {
		return err
	}
	if m.Version, err = dec.Uint8(); err != nil {
		return err
	}
	if m.Setting, err = dec.Uint8(); err != nil {
		return err
	}
	if m.MessageID, err = dec.Int64(); err != nil {
		return err
	}
	if m.MessageSeq, err = dec.Uint32(); err != nil {
		return err
	}
	if m.ClientMsgNo, err = dec.String(); err != nil {
		return err
	}
	if m.Timestamp, err = dec.Int32(); err != nil {
		return err
	}
	if m.FromUID, err = dec.String(); err != nil {
		return err
	}
	if m.ChannelID, err = dec.String(); err != nil {
		return err
	}
	if m.ChannelType, err = dec.Uint8(); err != nil {
		return err
	}
	if m.Payload, err = dec.BinaryAll(); err != nil {
		return err
	}
	return nil
}
