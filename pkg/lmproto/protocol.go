package lmproto

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/tangtaoit/limnet/pkg/limlog"
)

// Protocol Protocol
type Protocol interface {
	// DecodePacket 解码消息
	DecodePacket(data []byte, version uint8) (Frame, int, error)
	// EncodePacket 编码消息
	EncodePacket(packet interface{}, version uint8) ([]byte, error)
}

// LiMaoProto 狸猫协议对象
type LiMaoProto struct {
	limlog.Log
}

// LatestVersion 最新版本
const LatestVersion = 4

// MaxRemaingLength 最大剩余长度 // 1<<28 - 1
const MaxRemaingLength uint32 = 1024 * 1024

// New 创建limao协议对象
func New() *LiMaoProto {
	return &LiMaoProto{
		limlog.NewLIMLog("LiMaoProto"),
	}
}

// PacketDecodeFunc 包解码函数
type PacketDecodeFunc func(frame Frame, remainingBytes []byte, version uint8) (Frame, error)

// PacketEncodeFunc 包编码函数
type PacketEncodeFunc func(frame Frame, version uint8) ([]byte, error)

var packetDecodeMap = map[PacketType]PacketDecodeFunc{
	CONNECT:    decodeConnect,
	CONNACK:    decodeConnack,
	SEND:       decodeSend,
	SENDACK:    decodeSendack,
	RECV:       decodeRecv,
	RECVACK:    decodeRecvack,
	DISCONNECT: decodeDisConnect,
}
var packetEncodeMap = map[PacketType]PacketEncodeFunc{
	CONNECT:    encodeConnect,
	CONNACK:    encodeConnack,
	SEND:       encodeSend,
	SENDACK:    encodeSendack,
	RECV:       encodeRecv,
	RECVACK:    encodeRecvack,
	DISCONNECT: encodeDisConnect,
}

// DecodePacketWithConn 解码包
func (l *LiMaoProto) DecodePacketWithConn(conn io.Reader, version uint8) (Frame, error) {
	framer, err := l.decodeFramerWithConn(conn)
	if err != nil {
		return nil, err
	}
	// l.Debug("解码消息！", zap.String("framer", framer.String()))
	if framer.GetPacketType() == PING {
		return &PingPacket{}, nil
	}
	if framer.GetPacketType() == PONG {
		return &PongPacket{}, nil
	}

	if framer.RemainingLength > MaxRemaingLength {
		//return nil,errors.New(fmt.Sprintf("消息超出最大限制[%d]！",MaxRemaingLength))
		panic(errors.New(fmt.Sprintf("消息超出最大限制[%d]！", MaxRemaingLength)))
	}

	body := make([]byte, framer.RemainingLength)
	_, err = io.ReadFull(conn, body)
	if err != nil {
		return nil, err
	}
	decodeFunc := packetDecodeMap[framer.GetPacketType()]
	if decodeFunc == nil {
		return nil, errors.New(fmt.Sprintf("不支持对[%s]包的解码！", framer.GetPacketType()))
	}

	frame, err := decodeFunc(framer, body, version)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("解码包[%s]失败！", framer.GetPacketType()))
	}
	return frame, nil
}

// DecodePacket 解码包
func (l *LiMaoProto) DecodePacket(data []byte, version uint8) (Frame, int, error) {
	framer, remainingLengthLength, err := l.decodeFramer(data)
	if err != nil {
		return nil, 0, err
	}
	// l.Debug("解码消息！", zap.String("framer", framer.String()))
	if framer.GetPacketType() == PING {
		return &PingPacket{}, 1, nil
	}
	if framer.GetPacketType() == PONG {
		return &PongPacket{}, 1, nil
	}

	if framer.RemainingLength > MaxRemaingLength {
		//return nil,errors.New(fmt.Sprintf("消息超出最大限制[%d]！",MaxRemaingLength))
		return nil, 0, fmt.Errorf("消息超出最大限制[%d]！", MaxRemaingLength)
	}
	msgLen := int(framer.RemainingLength) + 1 + remainingLengthLength
	if len(data) < msgLen {
		return nil, 0, nil
	}
	body := data[1+remainingLengthLength : msgLen]
	decodeFunc := packetDecodeMap[framer.GetPacketType()]
	if decodeFunc == nil {
		return nil, 0, errors.New(fmt.Sprintf("不支持对[%s]包的解码！", framer.GetPacketType()))
	}

	frame, err := decodeFunc(framer, body, version)
	if err != nil {
		return nil, 0, errors.Wrap(err, fmt.Sprintf("解码包[%s]失败！", framer.GetPacketType()))
	}
	return frame, 1 + remainingLengthLength + int(framer.RemainingLength), nil
}

// EncodePacket 编码包
func (l *LiMaoProto) EncodePacket(packet interface{}, version uint8) ([]byte, error) {
	var frame = packet.(Frame)
	packetType := frame.GetPacketType()

	var bodyBytes []byte

	enc := NewEncoder()

	if frame.GetPacketType() != PING && frame.GetPacketType() != PONG {
		packetEncodeFunc := packetEncodeMap[packetType]
		if packetEncodeFunc == nil {
			return nil, errors.New(fmt.Sprintf("不支持对[%s]包的编码！", packetType))
		}
		bodyBytes, err := packetEncodeFunc(frame, version)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("编码包[%s]失败！", frame.GetPacketType()))
		}
		// FixedHeader
		headerBytes, err := l.encodeFramer(frame, uint32(len(bodyBytes)))
		if err != nil {
			return nil, err
		}
		enc.WriteBytes(headerBytes)
		enc.WriteBytes(bodyBytes)
	} else {
		// FixedHeader
		headerBytes, err := l.encodeFramer(frame, uint32(len(bodyBytes)))
		if err != nil {
			return nil, err
		}
		enc.WriteBytes(headerBytes)
	}
	return enc.Bytes(), nil
}

func (l *LiMaoProto) encodeFramer(f Frame, remainingLength uint32) ([]byte, error) {
	if f.GetPacketType() == PING || f.GetPacketType() == PONG {
		return []byte{byte(int(f.GetPacketType()<<4) | 0)}, nil
	}

	header := []byte{ToFixHeaderUint8(f)}

	if f.GetPacketType() == PING || f.GetPacketType() == PONG {
		return header, nil
	}
	varHeader := encodeVariable(remainingLength)

	return append(header, varHeader...), nil
}
func (l *LiMaoProto) decodeFramer(data []byte) (Framer, int, error) {
	typeAndFlags := data[0]
	p := FramerFromUint8(typeAndFlags)
	var remainingLengthLength uint32 = 0 // 剩余长度的长度
	if p.PacketType != PING && p.PacketType != PONG {
		p.RemainingLength, remainingLengthLength = decodeLength(data[1:])
	}
	return p, int(remainingLengthLength), nil
}

func (l *LiMaoProto) decodeFramerWithConn(conn io.Reader) (Framer, error) {
	b := make([]byte, 1)
	_, err := io.ReadFull(conn, b)
	if err != nil {
		return Framer{}, err
	}
	typeAndFlags := b[0]
	p := Framer{}
	p.NoPersist = (typeAndFlags & 0x01) > 0
	p.RedDot = (typeAndFlags >> 1 & 0x01) > 0
	p.SyncOnce = (typeAndFlags >> 2 & 0x01) > 0
	p.DUP = (typeAndFlags >> 3 & 0x01) > 0
	p.PacketType = PacketType(typeAndFlags >> 4)
	if p.PacketType != PING && p.PacketType != PONG {
		p.RemainingLength = uint32(decodeLengthWithConn(conn))
	}
	return p, nil
}

func encodeVariable(size uint32) []byte {
	ret := make([]byte, 0)
	for size > 0 {
		digit := byte(size % 0x80)
		size /= 0x80
		if size > 0 {
			digit |= 0x80
		}
		ret = append(ret, digit)
	}
	return ret
}
func decodeLength(data []byte) (uint32, uint32) {
	var rLength uint32
	var multiplier uint32
	offset := 0
	for multiplier < 27 { //fix: Infinite '(digit & 128) == 1' will cause the dead loop
		digit := data[offset]
		rLength |= uint32(digit&127) << multiplier
		if (digit & 128) == 0 {
			break
		}
		multiplier += 7
		offset++
	}
	return rLength, uint32(offset + 1)
}
func decodeLengthWithConn(r io.Reader) int {
	var rLength uint32
	var multiplier uint32
	b := make([]byte, 1)
	for multiplier < 27 { //fix: Infinite '(digit & 128) == 1' will cause the dead loop
		io.ReadFull(r, b)
		digit := b[0]
		rLength |= uint32(digit&127) << multiplier
		if (digit & 128) == 0 {
			break
		}
		multiplier += 7
	}
	return int(rLength)
}

func encodeBool(b bool) (i int) {
	if b {
		i = 1
	}
	return
}
