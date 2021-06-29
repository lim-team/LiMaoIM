package lim

import (
	"fmt"

	"github.com/lim-team/LiMaoIM/pkg/limlog"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/tangtaoit/limnet"
	"github.com/tangtaoit/limnet/pkg/limutil/sync/atomic"
	"go.uber.org/zap"
)

// Client 客户端
type Client struct {
	conn        limnet.Conn
	uid         string              // 用户UID
	deviceFlag  lmproto.DeviceFlag  // 设备标示
	deviceLevel lmproto.DeviceLevel // 设备等级
	l           *LiMao
	limlog.Log
	aesKey string // 消息密钥（用于加解密消息的）
	aesIV  string

	// ---------- 统计相关 ----------
	sendPacketCount atomic.Int64 // 发送包数量
	recvPacketCount atomic.Int64 // 接受包数量
	sendDataSize    atomic.Int64 // 发送数据大小
	recvDataSize    atomic.Int64 // 接受数据大小
}

// NewClient 创建一个新的客户端
func NewClient(uid string, deviceFlag lmproto.DeviceFlag, deviceLevel lmproto.DeviceLevel, conn limnet.Conn, aesKey, aesIV string, l *LiMao) *Client {
	return &Client{
		uid:         uid,
		deviceFlag:  deviceFlag,
		deviceLevel: deviceLevel,
		aesKey:      aesKey,
		aesIV:       aesIV,
		conn:        conn,
		l:           l,
		Log:         limlog.NewLIMLog(fmt.Sprintf("Client[%d]", conn.GetID())),
	}
}

// GetID GetID
func (c *Client) GetID() int64 {
	return c.conn.GetID()
}

func (c *Client) Write(data []byte) error {
	c.recvPacketCount.Add(1)
	c.recvDataSize.Add(len(data))
	c.l.monitor.DownstreamAdd(len(data))
	return c.conn.Write(data)
}

// Close Close
func (c *Client) Close() error {
	return c.conn.Close()
}

// Version Version
func (c *Client) Version() uint8 {
	return c.conn.Version()
}

// WritePacket 写包
func (c *Client) WritePacket(packet interface{}) {
	data, err := c.l.protocol.EncodePacket(packet, c.conn.Version())
	if err != nil {
		c.Warn("写包失败！", zap.Error(err))
		return
	}
	c.l.monitor.DownstreamPacketInc()
	if len(data) > 0 {
		c.Write(data)
	}
}

// 加密是否开启
func (c *Client) encryptionOn() bool {
	return len(c.aesKey) > 0
}

// 客户端收到数据大小
func (c *Client) clientSendDataSize(size int) {
	c.sendDataSize.Add(size)
}

// 客户端收到包数量递增
func (c *Client) clientSendPacketInc() {
	c.sendPacketCount.Add(1)
}

func (c *Client) String() string {
	return fmt.Sprintf("id: %d uid: %s deviceFlag:%s", c.GetID(), c.uid, c.deviceFlag.String())
}
