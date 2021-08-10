package lim

import (
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/tangtaoit/limnet"
)

func limUnpacket(c limnet.Conn) ([]byte, error) {
	buff := c.Read()
	if len(buff) <= 0 {
		return nil, nil
	}
	offset := 0
	packetData := make([]byte, 0, len(buff))
	for len(buff) > offset {
		typeAndFlags := buff[offset]
		packetType := lmproto.PacketType(typeAndFlags >> 4)
		if packetType == lmproto.PING || packetType == lmproto.PONG {
			packetData = append(packetData, buff[offset])
			offset++
			continue
		}
		reminLen, readSize, has := decodeLength(buff[offset+1:])
		if !has {
			break
		}
		dataStart := offset
		dataEnd := offset + readSize + reminLen + 1
		if len(buff) >= dataEnd { // 总数据长度大于当前包数据长度 说明还有包可读。
			data := buff[dataStart:dataEnd]
			packetData = append(packetData, data...)
			offset = dataEnd
			continue
		} else {
			break
		}
	}

	if len(packetData) > 0 {
		c.ShiftN(len(packetData))
		return packetData, nil
	}

	return nil, nil
}

func decodeLength(data []byte) (int, int, bool) {
	var rLength uint32
	var multiplier uint32
	offset := 0
	for multiplier < 27 { //fix: Infinite '(digit & 128) == 1' will cause the dead loop
		if offset >= len(data) {
			return 0, 0, false
		}
		digit := data[offset]
		offset++
		rLength |= uint32(digit&127) << multiplier
		if (digit & 128) == 0 {
			break
		}
		multiplier += 7
	}
	return int(rLength), offset, true
}
