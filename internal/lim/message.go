package lim

import (
	"fmt"

	"github.com/lim-team/LiMaoIM/pkg/lmproto"
)

// Message 消息对象
type Message struct {
	lmproto.RecvPacket
	ToUID          string             // 接受者
	Subscribers    []string           // 订阅者 如果此字段有值 则表示消息只发送给指定的订阅者
	fromDeviceFlag lmproto.DeviceFlag // 发送者设备标示
	// 重试相同的clientID
	toClientID int64 // 指定接收客户端的ID
	// ------- 优先队列用到 ------
	index int   //在切片中的索引值
	pri   int64 // 优先级的时间点 值越小越优先
}

func (m *Message) String() string {
	return fmt.Sprintf("%s ToUID: %s Subscribers: %s  fromDeviceFlag:%s toClientID: %d", m.RecvPacket.String(), m.ToUID, m.Subscribers, m.fromDeviceFlag.String(), m.toClientID)
}
