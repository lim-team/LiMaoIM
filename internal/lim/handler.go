package lim

import (
	"encoding/base64"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/lim-team/LiMaoIM/internal/db"
	"github.com/lim-team/LiMaoIM/internal/lim/rpc"
	"github.com/lim-team/LiMaoIM/pkg/limlog"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/lmproxyproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"github.com/pkg/errors"
	"github.com/tangtaoit/limnet"
	"go.uber.org/zap"
)

// PacketHandler PacketHandler
type PacketHandler struct {
	l *LiMao
	limlog.Log
	messageIDGen *snowflake.Node // 消息ID生成器
}

// NewPacketHandler 创建处理者
func NewPacketHandler(l *LiMao) *PacketHandler {
	h := &PacketHandler{
		l:   l,
		Log: limlog.NewLIMLog("Handler"),
	}
	// Initialize the messageID generator of the snowflake algorithm
	var err error
	h.messageIDGen, err = snowflake.NewNode(int64(l.opts.NodeID))
	if err != nil {
		panic(err)
	}
	return h
}

// 生成消息ID
func (s *PacketHandler) genMessageID() int64 {
	return s.messageIDGen.Generate().Int64()
}

func (s *PacketHandler) handlePing(c *Client) {
	c.WritePacket(&lmproto.PongPacket{})
}

// 处理收到消息ack
func (s *PacketHandler) handleRecvack(c *Client, recvackPacket *lmproto.RecvackPacket) {
	c.Debug("收到回复包。", zap.String("packet", recvackPacket.String()))
	// 完成消息（移除重试队列里的消息）
	err := s.l.retryQueue.finishMessage(c.GetID(), recvackPacket.MessageID)
	if err != nil {
		c.Warn("移除重试队列里的消息失败！", zap.Error(err), zap.Int64("clientID", c.GetID()), zap.Uint8("deviceFlag", c.deviceFlag.ToUint8()), zap.Int64("messageID", recvackPacket.MessageID))
	}
	if recvackPacket.SyncOnce && !recvackPacket.NoPersist {
		err = s.l.store.UpdateMessageOfUserCursorIfNeed(c.uid, recvackPacket.MessageSeq)
		if err != nil {
			c.Warn("更新游标失败！", zap.Error(err), zap.String("uid", c.uid), zap.Uint32("messageSeq", recvackPacket.MessageSeq))
		}
	}
}

// 处理连接
func (s *PacketHandler) handleConnect(c limnet.Conn, connectPacket *lmproto.ConnectPacket) {
	if s.l.opts.IsCluster {
		allocNode := s.l.clusterManager.GetNode(connectPacket.UID) // Get the node where the user is
		if allocNode == nil {
			s.Warn("No node is obtained", zap.String("uid", connectPacket.UID))
			c.Close()
			return
		}
		if allocNode.NodeID != s.l.opts.NodeID {
			s.Warn("The user is not on this node", zap.Any("shouldNode", allocNode))
			s.writeConnack(c, 0, lmproto.ReasonUserNotOnNode)
			return
		}
	}
	c.SetVersion(connectPacket.Version) // Set the protocol version of the current client
	s.Debug("Connection package received", zap.Any("packet", connectPacket))
	timeDiff := time.Now().UnixNano()/1000/1000 - connectPacket.ClientTimestamp
	var deviceLevel lmproto.DeviceLevel = lmproto.DeviceLevelMaster
	if s.l.opts.Mode != TestMode { // test mode
		// ---------- token ----------
		var token string
		var err error
		token, deviceLevel, err = s.l.store.GetUserToken(connectPacket.UID, connectPacket.DeviceFlag)
		if err != nil {
			s.Error("Failed to query the user's token", zap.Error(err), zap.String("packet", connectPacket.String()))
			s.writeConnackError(c)
			return
		}
		// token不匹配
		if token != connectPacket.Token {
			s.Error("The token does not match, the connection failed", zap.String("packet", connectPacket.String()))
			s.writeConnackAuthFail(c)
			return
		}
	}
	var aesKey, aesIV string        // aes encrypted key and iv
	var dhServerPublicKeyEnc string // DH public key of the server
	if connectPacket.Version > 2 {
		clientKeyBytes, err := base64.StdEncoding.DecodeString(connectPacket.ClientKey)
		if err != nil {
			s.Error("Failed to decode the client's key", zap.String("packet", connectPacket.String()))
			s.writeConnackAuthFail(c)
			return
		}
		dhServerPrivKey, dhServerPublicKey := util.GetCurve25519KeypPair() // 生成服务器的DH密钥对

		var dhClientPubKeyArray [32]byte
		copy(dhClientPubKeyArray[:], clientKeyBytes[:32])
		shareKey := util.GetCurve25519Key(dhServerPrivKey, dhClientPubKeyArray) // 共享key

		aesIV = util.GetRandomString(16)
		aesKey = util.MD5(base64.StdEncoding.EncodeToString(shareKey[:]))[:16]

		dhServerPublicKeyEnc = base64.StdEncoding.EncodeToString(dhServerPublicKey[:])
	}
	oldClient := s.l.clientManager.GetClientWith(connectPacket.UID, connectPacket.DeviceFlag)
	if oldClient != nil { // There is an old connection, send a disconnect message first, and then close the client's connection
		s.l.clientManager.Remove(oldClient.GetID())
		disconnectData, _ := s.l.protocol.EncodePacket(&lmproto.DisconnectPacket{
			ReasonCode: 0,
			Reason:     "Account login on other devices",
		}, connectPacket.Version)
		oldClient.conn.Write(disconnectData)
		s.l.timingWheel.AfterFunc(time.Second*4, func() {
			oldClient.conn.Close() // Close old connection
		})
		s.Debug("Close old client", zap.Any("oldClient", oldClient))
	}
	c.SetStatus(ConnStatusAuthed.Int()) // Set the connection status to authenticated

	// add client
	client := NewClient(connectPacket.UID, connectPacket.DeviceFlag, deviceLevel, c, aesKey, aesIV, s.l)
	s.l.clientManager.Add(client)
	// Encoded connection receipt package
	client.WritePacket(&lmproto.ConnackPacket{
		Salt:       aesIV,
		ServerKey:  dhServerPublicKeyEnc,
		ReasonCode: lmproto.ReasonSuccess,
		TimeDiff:   timeDiff,
	})
	// 在线webhook
	s.l.onlineStatusWebhook.Online(connectPacket.UID, connectPacket.DeviceFlag)
}

// Handling client disconnects
func (s *PacketHandler) handleDisconnect(c limnet.Conn) {

	client := s.l.clientManager.Get(c.GetID())
	if client != nil {
		// Remove client
		s.l.clientManager.Remove(c.GetID())
		newClient := s.l.clientManager.GetClientWith(client.uid, client.deviceFlag) // 指定的uid和设备下没有新的客户端才算真真的下线（TODO: 有时候离线要必在线晚触发导致不正确）
		if newClient == nil {
			s.l.onlineStatusWebhook.Offline(client.uid, client.deviceFlag)
		}
	}

}

func (s *PacketHandler) writeConnackError(conn limnet.Conn) error {
	return s.writeConnack(conn, 0, lmproto.ReasonError)
}
func (s *PacketHandler) writeConnackAuthFail(conn limnet.Conn) error {
	return s.writeConnack(conn, 0, lmproto.ReasonAuthFail)
}
func (s *PacketHandler) writeConnack(conn limnet.Conn, timeDiff int64, code lmproto.ReasonCode) error {
	data, err := s.l.protocol.EncodePacket(&lmproto.ConnackPacket{
		ReasonCode: code,
		TimeDiff:   timeDiff,
	}, conn.Version())
	if err != nil {
		return err
	}
	s.l.monitor.DownstreamAdd(len(data))
	s.l.monitor.DownstreamPacketInc()
	conn.Write(data)
	return nil
}

// HandleSend HandleSend
func (s *PacketHandler) HandleSend(c *Client, sendPacket *lmproto.SendPacket) {
	// s.Debug("Received the message", zap.Any("packet", sendPacket))
	// c.WritePacket(&lmproto.SendackPacket{
	// 	ClientSeq:   sendPacket.ClientSeq,
	// 	ClientMsgNo: sendPacket.ClientMsgNo,
	// 	ReasonCode:  lmproto.ReasonSuccess,
	// })
	// return
	var messageID = s.genMessageID()
	if c.conn.Version() > 2 {
		signStr := sendPacket.VerityString()
		actMsgKey, err := util.AesEncryptPkcs7Base64([]byte(signStr), []byte(c.aesKey), []byte(c.aesIV))
		if err != nil {
			s.Error("MsgKey is illegal！", zap.Error(err))
			c.WritePacket(&lmproto.SendackPacket{
				ClientSeq:   sendPacket.ClientSeq,
				ClientMsgNo: sendPacket.ClientMsgNo,
				MessageID:   messageID,
				ReasonCode:  lmproto.ReasonMsgKeyError,
			})
			return
		}

		actMsgKeyStr := sendPacket.MsgKey

		exceptMsgKey := util.MD5(string(actMsgKey))

		if actMsgKeyStr != exceptMsgKey {
			s.Error("MsgKey is illegal！", zap.String("except", exceptMsgKey), zap.String("act", actMsgKeyStr))
			c.WritePacket(&lmproto.SendackPacket{
				ClientSeq:   sendPacket.ClientSeq,
				ClientMsgNo: sendPacket.ClientMsgNo,
				MessageID:   messageID,
				ReasonCode:  lmproto.ReasonMsgKeyError,
			})
			return
		}
		decodePayload, err := util.AesDecryptPkcs7Base64(sendPacket.Payload, []byte(c.aesKey), []byte(c.aesIV))
		if err != nil {
			s.Error("Failed to decode payload！", zap.Error(err))
			c.WritePacket(&lmproto.SendackPacket{
				ClientSeq:   sendPacket.ClientSeq,
				ClientMsgNo: sendPacket.ClientMsgNo,
				MessageID:   messageID,
				ReasonCode:  lmproto.ReasonPayloadDecodeError,
			})
			return
		}
		sendPacket.Payload = decodePayload
	}
	if !s.l.opts.IsCluster {
		// 处理属于本节点的发送包
		s.handleLocalSend(c, sendPacket)
		return
	}
	allocNode := s.l.clusterManager.GetNode(c.uid) // 获取分配的节点
	if allocNode == nil {
		s.Error("没有获取到节点，关闭连接！")
		return
	}
	if allocNode.NodeID != s.l.opts.NodeID {
		s.Warn("用户没在此节点上，不能发送消息！将断开连接！", zap.Any("shouldNode", allocNode))
		c.conn.Close()
		return
	}
	fakeChannelID := sendPacket.ChannelID
	if sendPacket.ChannelType == ChannelTypePerson {
		fakeChannelID = GetFakeChannelIDWith(c.uid, sendPacket.ChannelID)
	}
	node := s.l.clusterManager.GetNode(fakeChannelID) // 获取接受频道所在节点
	if node == nil {
		s.Error("没有获取到节点，关闭连接！")
		return
	}
	/**
	如果频道在当前节点，则先存储频道消息，然后获取频道所有订阅者，找到订阅者们所属的各个节点，然后将消息转发给对应的节点
	如果不在当前节点，则转发给对应的节点
	**/
	if node.NodeID != s.l.opts.NodeID {
		packetData, _ := c.l.protocol.EncodePacket(sendPacket, lmproto.LatestVersion)
		req := &rpc.ForwardSendPacketReq{
			SendPacket:     packetData,
			FromUID:        c.uid,
			FromDeviceFlag: int32(c.deviceFlag),
		}
		// 转发消息到对应的节点
		resp, err := s.l.nodeRemoteCall.ForwardSendPacket(req, node.NodeID)
		if err != nil {
			s.Error("转发消息失败！", zap.Error(err), zap.Any("node", node))
			c.WritePacket(&lmproto.SendackPacket{
				ClientSeq:   sendPacket.ClientSeq,
				ClientMsgNo: sendPacket.ClientMsgNo,
				MessageID:   messageID,
				ReasonCode:  lmproto.ReasonError,
			})
			return
		}
		c.WritePacket(&lmproto.SendackPacket{
			ClientSeq:   sendPacket.ClientSeq,
			ClientMsgNo: sendPacket.ClientMsgNo,
			MessageID:   resp.MessageID,
			MessageSeq:  uint32(resp.MessageSeq),
			ReasonCode:  lmproto.ReasonCode(resp.ReasonCode),
		})
		return
	}
	// 处理属于本节点的发送包
	s.handleLocalSend(c, sendPacket)
}

func (s *PacketHandler) handleLocalSend(c *Client, sendPacket *lmproto.SendPacket) {
	messageID, messageSeq, reasonCode, _ := s.handleSendPacketWithFrom(c.uid, c.deviceFlag, sendPacket)
	c.WritePacket(&lmproto.SendackPacket{
		ClientSeq:   sendPacket.ClientSeq,
		ClientMsgNo: sendPacket.ClientMsgNo,
		MessageID:   messageID,
		MessageSeq:  messageSeq,
		ReasonCode:  reasonCode,
	})
}

func (s *PacketHandler) allowSendOfPersonChannel(fromUID, toUID string) (bool, error) {
	fromChannel, err := s.l.channelManager.GetChannel(fromUID, ChannelTypePerson)
	if err != nil {
		return false, errors.New("获取发送者的频道失败！")
	}
	if fromChannel != nil {
		allow := fromChannel.Allow(toUID)
		return allow, nil
	}
	return false, nil
}

func (s *PacketHandler) handleSendPacketWithFrom(fromUID string, fromDeviceFlag lmproto.DeviceFlag, sendPacket *lmproto.SendPacket) (int64, uint32, lmproto.ReasonCode, error) {
	var allocNode *lmproxyproto.Node // 获取分配的节点
	if s.l.opts.IsCluster {
		allocNode = s.l.clusterManager.GetNode(fromUID) // 获取分配的节点
	}
	if s.l.opts.IsCluster && allocNode == nil {
		s.Error("没有获取到节点，关闭连接！", zap.String("fromUID", fromUID))
		return 0, 0, lmproto.ReasonUserNotOnNode, errors.New("没有获取到节点，关闭连接！")
	}
	fakeChannelID := sendPacket.ChannelID
	if sendPacket.ChannelType == ChannelTypePerson {
		fakeChannelID = GetFakeChannelIDWith(fromUID, sendPacket.ChannelID)
		allow, err := s.allowSendOfPersonChannel(fromUID, sendPacket.ChannelID)
		if err != nil {
			s.Error("判断个人频道是否允许发送消息失败！", zap.Error(err))
			return 0, 0, lmproto.ReasonError, errors.New("判断个人频道是否允许发送消息失败！")
		}
		if !allow {
			s.Error("允许给用户发送消息", zap.String("fromUID", fromUID), zap.String("toUID", sendPacket.ChannelID))
			return 0, 0, lmproto.ReasonNotAllowSend, errors.New("不允许发送!")
		}
	}

	var messageID = s.genMessageID()
	// 获取发送消息的频道
	channel, reasonCode, err := s.getSendChannel(fromUID, fakeChannelID, sendPacket.ChannelType)
	if err != nil {
		s.Error("Failed to get sending channel", zap.Error(err))
		return messageID, 0, reasonCode, err
	}
	if channel == nil {
		return messageID, 0, lmproto.ReasonSubscriberNotExist, nil
	}
	var messageSeq uint32
	if !sendPacket.GetNoPersist() && !sendPacket.SyncOnce { // Only messages that need to be stored will increment the sequence number
		messageSeq, err = s.l.store.GetNextMessageSeq(fakeChannelID, sendPacket.ChannelType)
		if err != nil {
			s.Error("获取频道消息序列号失败！", zap.String("channelID", fakeChannelID), zap.Uint8("channelType", sendPacket.ChannelType), zap.Error(err))
			return messageID, messageSeq, lmproto.ReasonError, err
		}
	}
	// 转换成接受包
	recvPacket := &lmproto.RecvPacket{
		Framer: lmproto.Framer{
			RedDot:    sendPacket.GetRedDot(),
			SyncOnce:  sendPacket.GetsyncOnce(),
			NoPersist: sendPacket.GetNoPersist(),
		},
		Setting:     sendPacket.Setting,
		MessageID:   messageID,
		MessageSeq:  messageSeq,
		ClientMsgNo: sendPacket.ClientMsgNo,
		FromUID:     fromUID,
		ChannelID:   sendPacket.ChannelID,
		ChannelType: sendPacket.ChannelType,
		Timestamp:   int32(time.Now().Unix()),
		Payload:     sendPacket.Payload,
	}

	messageD := &db.Message{
		Header:      lmproto.ToFixHeaderUint8(recvPacket),
		Setting:     recvPacket.Setting.ToUint8(),
		MessageID:   recvPacket.MessageID,
		MessageSeq:  recvPacket.MessageSeq,
		ClientMsgNo: recvPacket.ClientMsgNo,
		Timestamp:   recvPacket.Timestamp,
		FromUID:     recvPacket.FromUID,
		ChannelID:   fakeChannelID,
		ChannelType: recvPacket.ChannelType,
		Payload:     recvPacket.Payload,
	}
	if !sendPacket.NoPersist && !sendPacket.SyncOnce { // If it is SyncOnce, it will not be stored, because SyncOnce messages are stored in the user’s respective queues.
		err = s.l.store.AppendMessage(messageD)
		if err != nil {
			s.Error("Failed to save history message", zap.Error(err))
			return messageID, messageSeq, lmproto.ReasonError, err
		}
	}

	if s.l.opts.WebhookOn() {
		// Add a message to the notification queue, the data in this queue will be notified to third-party applications
		err = s.l.store.AppendMessageOfNotifyQueue(messageD)
		if err != nil {
			s.Error("添加消息到通知队列失败！", zap.Error(err))
			return messageID, messageSeq, lmproto.ReasonError, err
		}
	}
	message := &Message{
		RecvPacket:     *recvPacket,
		fromDeviceFlag: fromDeviceFlag,
	}
	// // 处理消息
	err = channel.PutMessage(message)
	if err != nil {
		s.Error("将消息放入频道内失败！", zap.Error(err))
		return messageID, messageSeq, lmproto.ReasonError, err
	}
	return messageID, messageSeq, lmproto.ReasonSuccess, nil
}

// 获取发送频道
func (s *PacketHandler) getSendChannel(fromUID string, channelID string, channelType uint8) (*Channel, lmproto.ReasonCode, error) {
	channel, err := s.l.channelManager.GetChannel(channelID, channelType)
	if err != nil {
		s.Error("Failed to get channel", zap.Error(err))
		return nil, lmproto.ReasonError, err
	}
	if channel == nil {
		s.Error("The channel does not exist or has been disbanded", zap.String("channel_id", channelID), zap.Uint8("channel_type", channelType))
		return nil, lmproto.ReasonError, nil
	}
	if s.l.opts.Mode != TestMode {
		if !channel.Allow(fromUID) {
			s.Error("The user is not in the white list or in the black list", zap.String("fromUID", fromUID), zap.Error(err))
			return nil, lmproto.ReasonNotAllowSend, nil
		}
		if channel.ChannelType != ChannelTypePerson {
			if !channel.IsSubscriber(fromUID) {
				s.Error("The user is not in the channel and cannot send messages to the channel", zap.String("fromUID", fromUID), zap.String("channel_id", channelID), zap.Uint8("channel_type", channelType), zap.Error(err))
				return nil, lmproto.ReasonNotAllowSend, nil
			}
		}
	}
	return channel, lmproto.ReasonSuccess, nil
}

//########## 其他节点调用 ##########

// OnSendPacket 其他服务端节点发送过来的包
func (s *PacketHandler) OnSendPacket(fromUID string, deviceFlag lmproto.DeviceFlag, sendPacket *lmproto.SendPacket) (messageID int64, messageSeq uint32, reasonCode lmproto.ReasonCode, err error) {
	s.Debug("收到转发发送包->", zap.String("fromUID", fromUID), zap.Any("packet", sendPacket))
	return s.handleSendPacketWithFrom(fromUID, deviceFlag, sendPacket)
}

// OnRecvPacket 其他服务端节点发送过来的接受包
func (s *PacketHandler) OnRecvPacket(deviceFlag lmproto.DeviceFlag, recvPacket *lmproto.RecvPacket, users []string) error {
	s.Debug("收到转发接受包->", zap.Any("packet", recvPacket), zap.Strings("subscribers", users))
	message := &Message{
		RecvPacket:     *recvPacket,
		fromDeviceFlag: deviceFlag,
	}
	return s.l.handleLocalSubscribersMessage(message, users)
}

// OnGetSubscribers 其他节点获取订阅者
func (s *PacketHandler) OnGetSubscribers(channelID string, channelType uint8) ([]string, error) {
	channel, err := s.l.channelManager.GetChannel(channelID, channelType)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		s.Error("频道不存在！", zap.String("channelID", channelID), zap.Uint8("channelType", channelType))
		return nil, errors.New("频道不存在！")
	}
	return channel.GetAllSubscribers(), nil
}

// OnGetChannelMessageSeq OnGetChannelMessageSeq
func (s *PacketHandler) OnGetChannelMessageSeq(channelID string, channelType uint8) (uint32, error) {
	messageSeq, err := s.l.store.GetNextMessageSeq(channelID, channelType)
	if err != nil {
		return 0, err
	}
	return messageSeq, nil
}
