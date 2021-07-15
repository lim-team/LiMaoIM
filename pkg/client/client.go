package client

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lim-team/LiMaoIM/pkg/limlog"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"github.com/lim-team/LiMaoIM/pkg/wait"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// OnRecv 收到消息事件
type OnRecv func(recv *lmproto.RecvPacket) error

// OnSendack 发送消息回执
type OnSendack func(sendack *lmproto.SendackPacket)

// OnClose 连接关闭
type OnClose func()

// Client 狸猫客户端
type Client struct {
	opts              *Options              // 狸猫IM配置
	sending           []*lmproto.SendPacket // 发送中的包
	sendingLock       sync.RWMutex
	proto             *lmproto.LiMaoProto
	addr              string      // 连接地址
	connected         atomic.Bool // 是否已连接
	conn              net.Conn
	heartbeatTicker   *time.Ticker // 心跳定时器
	stopHeartbeatChan chan bool
	retryPingCount    int // 重试ping次数
	clientIDGen       atomic.Uint64
	onRecv            OnRecv
	onClose           OnClose
	onSendack         OnSendack
	sendTotalMsgBytes atomic.Int64 // 发送消息总bytes数
	recvMsgCount      atomic.Int64 // 收取消息数量
	sendFailMsgCount  atomic.Int64 // 发送失败的消息数量
	clientPrivKey     [32]byte
	limlog.Log
	aesKey         string // aes密钥
	salt           string // 安全码
	wait           wait.Wait
	disconnectChan chan struct{}
	rw             *bufio.ReadWriter
}

// New 创建客户端
func New(addr string, opts ...Option) *Client {
	var defaultOpts = NewOptions()
	for _, opt := range opts {
		if opt != nil {
			if err := opt(defaultOpts); err != nil {
				panic(err)
			}
		}
	}
	return &Client{
		opts:              defaultOpts,
		addr:              addr,
		sending:           make([]*lmproto.SendPacket, 0),
		proto:             lmproto.New(),
		heartbeatTicker:   time.NewTicker(time.Second * 20),
		stopHeartbeatChan: make(chan bool, 0),
		Log:               limlog.NewLIMLog(fmt.Sprintf("IMClient[%s]", defaultOpts.UID)),
		wait:              wait.New(),
		disconnectChan:    make(chan struct{}),
	}
}

// GetOptions GetOptions
func (c *Client) GetOptions() *Options {
	return c.opts
}

// Connect 连接到IM
func (c *Client) Connect() error {
	network, address, _ := parseAddr(c.addr)
	var err error
	c.conn, err = net.Dial(network, address)
	if err != nil {
		return err
	}
	c.rw = bufio.NewReadWriter(bufio.NewReader(c.conn), bufio.NewWriter(c.conn))
	var clientPubKey [32]byte
	c.clientPrivKey, clientPubKey = util.GetCurve25519KeypPair() // 生成服务器的DH密钥对

	err = c.sendPacket(&lmproto.ConnectPacket{
		Version:         c.opts.ProtoVersion,
		DeviceFlag:      lmproto.APP,
		ClientKey:       base64.StdEncoding.EncodeToString(clientPubKey[:]),
		ClientTimestamp: time.Now().Unix(),
		UID:             c.opts.UID,
		Token:           c.opts.Token,
	})
	if err != nil {
		return err
	}
	err = c.rw.Flush()
	if err != nil {
		return err
	}
	f, err := c.proto.DecodePacketWithConn(c.rw, c.opts.ProtoVersion)
	if err != nil {
		return err
	}
	connack, ok := f.(*lmproto.ConnackPacket)
	if !ok {
		return errors.New("返回包类型有误！不是连接回执包！")
	}
	if connack.ReasonCode != lmproto.ReasonSuccess {
		return errors.New("连接失败！")
	}
	if len(c.sending) > 0 {
		for _, packet := range c.sending {
			c.sendPacket(packet)
		}
	}
	c.salt = connack.Salt

	serverKey, err := base64.StdEncoding.DecodeString(connack.ServerKey)
	if err != nil {
		return err
	}
	var serverPubKey [32]byte
	copy(serverPubKey[:], serverKey[:32])

	shareKey := util.GetCurve25519Key(c.clientPrivKey, serverPubKey) // 共享key
	c.aesKey = util.MD5(base64.StdEncoding.EncodeToString(shareKey[:]))[:16]
	c.connected.Store(true)
	go c.loopConn()
	go c.loopPing()
	return nil
}

// Disconnect 断开IM
func (c *Client) Disconnect() {
	c.handleClose()
	close(c.disconnectChan)
}

func (c *Client) handleClose() {
	if c.connected.Load() {
		c.connected.Store(false)
		c.conn.Close()
		if c.onClose != nil {
			c.onClose()
		}
	}
}

// SendMessage 发送消息
func (c *Client) SendMessage(channel *Channel, payload []byte, opt ...SendOption) error {
	opts := NewSendOptions()
	if len(opt) > 0 {
		for _, op := range opt {
			op(opts)
		}
	}
	// 加密消息内容
	encPayload, err := util.AesEncryptPkcs7Base64(payload, []byte(c.aesKey), []byte(c.salt))
	if err != nil {
		c.Error("加密消息payload失败！", zap.Error(err))
		return err
	}

	packet := &lmproto.SendPacket{
		Framer: lmproto.Framer{
			NoPersist: opts.NoPersist,
			SyncOnce:  opts.SyncOnce,
			RedDot:    opts.RedDot,
		},
		ClientSeq:   c.clientIDGen.Add(1),
		ClientMsgNo: util.GenerUUID(),
		ChannelID:   channel.ChannelID,
		ChannelType: channel.ChannelType,
		Payload:     encPayload,
	}
	packet.RedDot = true

	// 加密消息通道
	signStr := packet.VerityString()
	actMsgKey, err := util.AesEncryptPkcs7Base64([]byte(signStr), []byte(c.aesKey), []byte(c.salt))
	if err != nil {
		c.Error("加密数据失败！", zap.Error(err))
		return err
	}
	packet.MsgKey = util.MD5(string(actMsgKey))

	c.sendingLock.Lock()
	c.sending = append(c.sending, packet)
	c.sendingLock.Unlock()
	err = c.sendPacket(packet)
	if err != nil {
		return err
	}
	if opts.Flush {
		return c.rw.Flush()
	}
	return nil
}

// Flush Flush
func (c *Client) Flush() error {
	return c.rw.Flush()
}

// SendMessageSync 同步发送
func (c *Client) SendMessageSync(ctx context.Context, channel *Channel, payload []byte) (*lmproto.SendackPacket, error) {
	// 加密消息内容
	encPayload, err := util.AesEncryptPkcs7Base64(payload, []byte(c.aesKey), []byte(c.salt))
	if err != nil {
		c.Error("加密消息payload失败！", zap.Error(err))
		return nil, err
	}

	packet := &lmproto.SendPacket{
		ClientSeq:   c.clientIDGen.Add(1),
		ClientMsgNo: util.GenerUUID(),
		ChannelID:   channel.ChannelID,
		ChannelType: channel.ChannelType,
		Payload:     encPayload,
	}
	packet.RedDot = true

	// 加密消息通道
	signStr := packet.VerityString()
	actMsgKey, err := util.AesEncryptPkcs7Base64([]byte(signStr), []byte(c.aesKey), []byte(c.salt))
	if err != nil {
		c.Error("加密数据失败！", zap.Error(err))
		return nil, err
	}
	packet.MsgKey = util.MD5(string(actMsgKey))

	resultChan := c.wait.Register(packet.ClientSeq)
	err = c.sendPacket(packet)
	if err != nil {
		c.wait.Trigger(packet.ClientSeq, nil)
		return nil, err
	}
	select {
	case result := <-resultChan:
		return result.(*lmproto.SendackPacket), nil
	case <-c.disconnectChan:
		return nil, nil
	case <-ctx.Done():
		return nil, errors.New("send timeout")
	}

}

// SetOnRecv 设置收消息事件
func (c *Client) SetOnRecv(onRecv OnRecv) {
	c.onRecv = onRecv
}

// SetOnClose 设置关闭事件
func (c *Client) SetOnClose(onClose OnClose) {
	c.onClose = onClose
}

// SetOnSendack 设置发送回执
func (c *Client) SetOnSendack(onSendack OnSendack) {
	c.onSendack = onSendack
}

// GetSendMsgBytes 获取已发送字节数
func (c *Client) GetSendMsgBytes() int64 {
	return c.sendTotalMsgBytes.Load()
}

// GetRecvMsgCount GetRecvMsgCount
func (c *Client) GetRecvMsgCount() int64 {
	return c.recvMsgCount.Load()
}

// GetSendFailMsgCount GetSendFailMsgCount
func (c *Client) GetSendFailMsgCount() int64 {
	return c.sendFailMsgCount.Load()
}

func (c *Client) loopPing() {
	for {
		select {
		case <-c.heartbeatTicker.C:
			if c.retryPingCount >= 3 {
				c.conn.Close() // 如果重试三次没反应就断开连接，让其重连
				return
			}
			c.ping()
			c.retryPingCount++
			break
		case <-c.stopHeartbeatChan:
			goto exit
		}
	}
exit:
}

func (c *Client) ping() {
	err := c.sendPacket(&lmproto.PingPacket{})
	if err != nil {
		c.Warn("Ping发送失败！", zap.Error(err))
	}
}

// 发送包
func (c *Client) sendPacket(packet lmproto.Frame) error {
	data, err := c.proto.EncodePacket(packet, c.opts.ProtoVersion)
	if err != nil {
		return err
	}
	c.sendTotalMsgBytes.Add(int64(len(data)))
	_, err = c.rw.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) loopConn() {

	for {
		frame, err := c.proto.DecodePacketWithConn(c.rw, c.opts.ProtoVersion)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			}
			if err == io.EOF {

			}
			goto exit
		}
		c.handlePacket(frame)
	}
exit:
	c.Debug("loopConn quit")
	c.handleClose()
	c.stopHeartbeatChan <- true

	if c.opts.AutoReconn {
		c.Connect()
	}

}

func (c *Client) handlePacket(frame lmproto.Frame) {
	c.retryPingCount = 0 // 只要收到消息则重置ping次数
	switch frame.GetPacketType() {
	case lmproto.SENDACK: // 发送回执
		c.handleSendackPacket(frame.(*lmproto.SendackPacket))
		break
	case lmproto.RECV: // 收到消息
		c.handleRecvPacket(frame.(*lmproto.RecvPacket))
		break
	}
}

func (c *Client) handleSendackPacket(packet *lmproto.SendackPacket) {
	c.sendingLock.Lock()
	defer c.sendingLock.Unlock()
	if packet.ReasonCode != lmproto.ReasonSuccess {
		c.sendFailMsgCount.Inc()
	}
	if c.wait.IsRegistered(packet.ClientSeq) {
		c.wait.Trigger(packet.ClientSeq, packet)
	}
	if c.onSendack != nil {
		c.onSendack(packet)
	}

	for i, sendPacket := range c.sending {
		if sendPacket.ClientSeq == packet.ClientSeq {
			c.sending = append(c.sending[:i], c.sending[i+1:]...)
			break
		}
	}
}

// 处理接受包
func (c *Client) handleRecvPacket(packet *lmproto.RecvPacket) {
	c.recvMsgCount.Inc()
	var err error
	if c.onRecv != nil {
		decodePayload, err := util.AesDecryptPkcs7Base64(packet.Payload, []byte(c.aesKey), []byte(c.salt))
		if err != nil {
			panic(err)
		}
		packet.Payload = decodePayload
		err = c.onRecv(packet)
	}
	if err == nil {
		c.sendPacket(&lmproto.RecvackPacket{
			Framer:     packet.Framer,
			MessageID:  packet.MessageID,
			MessageSeq: packet.MessageSeq,
		})
	}
}

func parseAddr(addr string) (network, address string, port int) {
	network = "tcp"
	address = strings.ToLower(addr)
	if strings.Contains(address, "://") {
		pair := strings.Split(address, "://")
		network = pair[0]
		address = pair[1]
		pair2 := strings.Split(address, ":")
		portStr := pair2[1]
		portInt64, _ := strconv.ParseInt(portStr, 10, 64)
		port = int(portInt64)
	}
	return
}

// Channel Channel
type Channel struct {
	ChannelID   string
	ChannelType uint8
}

// NewChannel 创建频道
func NewChannel(channelID string, channelType uint8) *Channel {
	return &Channel{
		ChannelID:   channelID,
		ChannelType: channelType,
	}
}
