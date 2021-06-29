package lmproto

// PongPacket pong包对ping的回应
type PongPacket struct {
	Frame
}

// GetPacketType 包类型
func (p *PongPacket) GetPacketType() PacketType {
	return PONG
}
