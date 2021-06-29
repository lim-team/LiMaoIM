package lmproto

// PingPacket ping包
type PingPacket struct {
	Frame
}

// GetPacketType 包类型
func (p *PingPacket) GetPacketType() PacketType {
	return PING
}
