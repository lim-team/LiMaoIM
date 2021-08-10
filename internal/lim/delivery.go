package lim

import (
	"github.com/lim-team/LiMaoIM/internal/db"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"go.uber.org/zap"
)

// 存储在消息队列内 如果需要
func (l *LiMao) storeMessageToUserQueueIfNeed(m *Message, subscribers []string) (map[string]uint32, error) {
	subscriberSeqMap := map[string]uint32{}
	for _, subscriber := range subscribers {
		if m.SyncOnce && !m.NoPersist {
			seq, err := l.store.GetUserNextMessageSeq(subscriber)
			if err != nil {
				return nil, err
			}

			subscriberSeqMap[subscriber] = seq

			newFramer := m.Framer
			if subscriber == m.FromUID { // 如果是自己则不显示红点
				newFramer.RedDot = false
			}
			messageD := &db.Message{
				Header:      lmproto.ToFixHeaderUint8(newFramer),
				Setting:     m.Setting.ToUint8(),
				MessageID:   m.MessageID,
				MessageSeq:  seq,
				ClientMsgNo: m.ClientMsgNo,
				Timestamp:   m.Timestamp,
				FromUID:     m.FromUID,
				QueueUID:    subscriber,
				ChannelID:   m.ChannelID,
				ChannelType: m.ChannelType,
				Payload:     m.Payload,
			}
			if m.ChannelType == ChannelTypePerson && m.ChannelID == m.ToUID {
				messageD.ChannelID = m.FromUID
			}
			err = l.store.AppendMessageOfUser(messageD)
			if err != nil {
				return nil, err
			}
		}
	}
	return subscriberSeqMap, nil
}

// ==================== 消息投递 ====================
func (l *LiMao) startDeliveryMsg(m *Message, subscriberSeqMap map[string]uint32, subscribers ...string) {

	l.deliveryMsgPool.Submit(func() {
		l.DeliveryMsg(m, subscriberSeqMap, subscribers...)
	})
}

// DeliveryMsg 投递消息
func (l *LiMao) DeliveryMsg(msg *Message, subscriberSeqMap map[string]uint32, subscribers ...string) {

	offlineSubscribers := make([]string, 0, len(subscribers)) // 离线订阅者

	l.Debug("需要投递的订阅者", zap.Strings("subscribers", subscribers))
	for _, subscriber := range subscribers {
		recvClients := l.getRecvClients(msg, subscriber)
		channelID := msg.ChannelID
		if msg.ChannelType == ChannelTypePerson && msg.ChannelID == subscriber {
			channelID = msg.FromUID
		}
		if recvClients != nil && len(recvClients) > 0 {
			l.Debug("接受客户端数量", zap.String("subscriber", subscriber), zap.Int("count", len(recvClients)))
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
				if len(subscriberSeqMap) > 0 {
					seq := subscriberSeqMap[subscriber]
					if seq != 0 {
						newMsg.MessageSeq = seq
					}
				}
				l.retryQueue.startInFlightTimeout(newMsg)

				// 发送消息
				recvPacket := &lmproto.RecvPacket{
					Framer: lmproto.Framer{
						RedDot:    newMsg.RedDot,
						SyncOnce:  newMsg.SyncOnce,
						NoPersist: newMsg.NoPersist,
					},
					Setting:     newMsg.Setting,
					ClientMsgNo: newMsg.ClientMsgNo,
					MessageID:   newMsg.MessageID,
					MessageSeq:  newMsg.MessageSeq,
					Timestamp:   newMsg.Timestamp,
					FromUID:     newMsg.FromUID,
					ChannelID:   channelID,
					ChannelType: newMsg.ChannelType,
					Payload:     newMsg.Payload,
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
		// 推送离线到上层应用
		l.TriggerEvent(&Event{
			Event: EventMsgOffline,
			Data: struct {
				MessageResp
				ToUIDs []string `json:"to_uids"`
			}{
				MessageResp: MessageResp{
					Header: MessageHeader{
						RedDot:    util.BoolToInt(msg.RedDot),
						SyncOnce:  util.BoolToInt(msg.SyncOnce),
						NoPersist: util.BoolToInt(msg.NoPersist),
					},
					Setting:     msg.Setting.ToUint8(),
					ClientMsgNo: msg.ClientMsgNo,
					MessageID:   msg.MessageID,
					MessageSeq:  msg.MessageSeq,
					FromUID:     msg.FromUID,
					ChannelID:   msg.ChannelID,
					ChannelType: msg.ChannelType,
					Timestamp:   msg.Timestamp,
					Payload:     msg.Payload,
				},
				ToUIDs: offlineSubscribers,
			},
		})
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

// ==================== 节点数据投递 ====================

func (l *LiMao) startDeliveryNodeData(data *NodeInFlightData) {
	l.nodeInFlightQueue.startInFlightTimeout(data) // 重新投递

	err := l.nodeRemoteCall.ForwardRecvPacket(data.Req, data.NodeID)
	if err != nil {
		l.Warn("请求grpc投递节点数据失败！", zap.Error(err))
		return
	}
	l.nodeInFlightQueue.finishMessage(data.No)
}
