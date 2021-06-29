package lim

import (
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"go.uber.org/zap"
)

// ==================== 消息投递 ====================
func (l *LiMao) startDeliveryMsg(m *Message, subscribers ...string) {

	l.deliveryMsgPool.Submit(func() {
		l.DeliveryMsg(m, subscribers...)
	})
}

// DeliveryMsg 投递消息
func (l *LiMao) DeliveryMsg(msg *Message, subscribers ...string) {

	offlineSubscribers := make([]string, 0, len(subscribers)) // 离线订阅者

	l.Debug("需要投递的订阅者", zap.Strings("subscribers", subscribers))
	for _, subscriber := range subscribers {
		recvClients := l.getRecvClients(msg, subscriber)
		channelID := msg.ChannelID
		if msg.ChannelType == ChannelTypePerson && msg.ChannelID == subscriber {
			channelID = msg.FromUID
		}
		if recvClients != nil && len(recvClients) > 0 {
			l.Debug("接受客户端数量", zap.String("UID", msg.ToUID), zap.Int("count", len(recvClients)))
			for _, recvClient := range recvClients {
				if msg.toClientID != 0 && recvClient.GetID() != msg.toClientID { // 如果指定的ToClientID则消息只发给指定的设备，非指定设备不投递
					l.Debug("不是指定的设备投递！", zap.Int64("toClientID", msg.toClientID), zap.Int64("recvClientID", recvClient.GetID()))
					continue
				}
				if msg.toClientID != 0 {
					l.Debug("是重试消息！", zap.String("msg", msg.String()), zap.Any("client", recvClient))
				}
				// 放入超时队列
				var newMsg = &Message{}
				*newMsg = *msg
				newMsg.ToUID = subscriber
				newMsg.toClientID = recvClient.GetID()
				l.retryQueue.startInFlightTimeout(newMsg)

				// 发送消息
				recvPacket := &lmproto.RecvPacket{
					Framer: lmproto.Framer{
						RedDot:    msg.RedDot,
						SyncOnce:  msg.SyncOnce,
						NoPersist: msg.NoPersist,
					},
					Setting:     msg.Setting,
					ClientMsgNo: msg.ClientMsgNo,
					MessageID:   msg.MessageID,
					MessageSeq:  msg.MessageSeq,
					Timestamp:   msg.Timestamp,
					FromUID:     msg.FromUID,
					ChannelID:   channelID,
					ChannelType: msg.ChannelType,
					Payload:     msg.Payload,
				}
				if recvClient.conn.Version() > 2 {

					// 加密payload
					payloadEnc, err := util.AesEncryptPkcs7Base64(recvPacket.Payload, []byte(recvClient.aesKey), []byte(recvClient.aesIV))
					if err != nil {
						l.Debug("加密payload失败！", zap.Error(err))
						continue
					}
					recvPacket.Payload = payloadEnc

					// 生成MsgKey
					msgKeyBytes, err := util.AesEncryptPkcs7Base64([]byte(recvPacket.VerityString()), []byte(recvClient.aesKey), []byte(recvClient.aesIV))
					if err != nil {
						l.Debug("生成MsgKey失败！", zap.Error(err))
						continue
					}
					recvPacket.MsgKey = util.MD5(string(msgKeyBytes))
				}

				recvClient.WritePacket(recvPacket)
			}
		} else { // 离线消息
			if msg.toClientID != 0 {
				err := l.retryQueue.finishMessage(msg.toClientID, msg.MessageID)
				if err != nil {
					l.Warn("finishMessage失败！", zap.Error(err), zap.Int64("toClientID", msg.toClientID), zap.Int64("messageID", msg.MessageID))
				}
			}
			if subscriber == msg.FromUID { // 自己发给自己的消息不触发离线事件
				continue
			}
			offlineSubscribers = append(offlineSubscribers, subscriber)
		}
	}
	if len(offlineSubscribers) > 0 {
		l.Debug("Offline subscribers", zap.Strings("offlineSubscribers", offlineSubscribers))
	}

}

// 获取接受消息的客户端
func (l *LiMao) getRecvClients(msg *Message, subscriber string) []*Client {
	toClients := l.clientManager.GetClientsWithUID(subscriber)
	// fromClients := l.clientManager.GetClientsWithUID(msg.FromUID)

	clients := make([]*Client, 0, len(toClients))
	if len(toClients) > 0 {
		for _, client := range toClients {
			if !l.clientIsSelf(client, msg) {
				clients = append(clients, client)
			}
		}
	}
	return clients
}

// 客户端是发送者自己
func (l *LiMao) clientIsSelf(client *Client, msg *Message) bool {
	if client.uid == msg.FromUID && client.deviceFlag == msg.fromDeviceFlag {
		return true
	}
	return false
}
