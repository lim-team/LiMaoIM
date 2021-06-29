package lim

import (
	"encoding/binary"

	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/tangtaoit/limnet"
)

func limUnpacket(c limnet.Conn) ([]byte, error) {
	buff := c.Read()
	if len(buff) <= 0 {
		return nil, nil
	}

	flag := buff[0]
	if flag == 0x02 { // 如果第一个字节为0x02则一定是mosproto 因为lmproto第一个字节一定不会是0x02
		if len(buff) < 16 { // 如果小于16则一个包的字节数一定不完整
			return nil, nil
		}
		bodyLen := int(binary.LittleEndian.Uint32(buff[1:5]))
		if len(buff) >= bodyLen+16 {
			c.ShiftN(bodyLen + 16)
			return buff[:bodyLen+16], nil
		}
	} else {
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
