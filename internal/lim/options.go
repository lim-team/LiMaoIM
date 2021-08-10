package lim

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/configor"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
)

const (
	// DebugMode indicates gin mode is debug.
	DebugMode = "debug"
	// ReleaseMode indicates gin mode is release.
	ReleaseMode = "release"
	// TestMode indicates gin mode is test.
	TestMode = "test"
)

// Options IM配置信息
type Options struct {
	NodeID                     int32            `env:"nodeID"` // node id < 1024
	Addr                       string           `env:"addr"`   // server addr (Internet address)
	WSAddr                     string           `env:"wsAddr"` // websocket listening address
	HTTPAddr                   string           `env:"httpAddr"`
	Mode                       string           `env:"mode"`         // mode
	NodeRaftAddr               string           `env:"nodeRaftAddr"` // node address IP:PORT example127.0.0.1:6000
	NodeAPIAddr                string           `env:"nodeAPIAddr"`  // 节点API地址 例如: http://127.0.0.1:1516
	NodeRPCAddr                string           `env:"nodeRPCAddr"`  // 节点rpc通讯地址，主要用来转发消息
	NodeTCPAddr                string           `env:"nodeTCPAddr"`  // 节点的TCP地址 对外公开，APP端长连接通讯
	DataDir                    string           `env:"dataDir"`
	Proxy                      string           `env:"proxy"` // proxy server addr （If there is no value, stand-alone mode）
	ElectionTicks              int              `env:"electionTicks"`
	ProxyTickPer               duration         `env:"proxyTickPer"`
	SlotCount                  int              `env:"slotCount"` // slot count
	SegmentMaxBytes            int64            `env:"segmentMaxBytes"`
	CMDSendTimeout             duration         `env:"cmdSendTimeout"`
	EventPoolSize              int              // 事件协程池大小
	MessagePoolSize            int              `env:"messagePoolSize"` // The size of the coroutine pool for processing packets
	DeliveryMsgPoolSize        int              `env:"deliveryMsgPoolSize"`
	MsgTimeout                 duration         `env:"msgTimeout"`          // Message sending timeout time, after this time it will try again
	TimeoutScanInterval        duration         `env:"timeoutScanInterval"` //  Message timeout queue scan frequency
	Proto                      lmproto.Protocol // protocol
	Webhook                    string           `env:"webhook"`                   // The URL address of the third party that receives the message notification
	MessageNotifyMaxCount      int              `env:"messageNotifyMaxCount"`     // Maximum number of notifications received each time
	MessageNotifyScanInterval  duration         `env:"messageNotifyScanInterval"` // 消息通知间隔
	TimingWheelTick            duration         // The time-round training interval must be 1ms or more
	TimingWheelSize            int64            // Time wheel size
	MaxMessagePerSecond        int              `env:"maxMessagePerSecond"` //Number of messages per second
	TmpChannelSuffix           string           // Temporary channel suffix
	CreateIfChannelNotExist    bool             // If the channel does not exist, whether to create it automatically
	Datasource                 string           `env:"datasource"`               // Source address of external data source
	ConversationSyncInterval   duration         `env:"conversationSaveInterval"` // How often to sync recent conversations
	ConversationSyncOnce       int              `env:"conversationSaveOnce"`     // When how many recent sessions have not been saved, save once
	ConversationOfUserMaxCount int              `env:"conversationMaxCount"`     //The maximum number of user conversation

	NodeRPCMsgTimeout          duration // 节点消息发送超时时间，超过这时间将重试
	NodeRPCTimeoutScanInterval duration // 节点之间调用RPC消息超时队列扫描频率
	IsCluster                  bool     // 是否开启分布式
	SlotBackupDir              string   // slot备份目录
}

// NewOptions NewOptions
func NewOptions() *Options {

	opts := &Options{
		NodeID:        1,
		ElectionTicks: 10,
		ProxyTickPer: duration{
			Duration: time.Second,
		},
		Addr:            "tcp://0.0.0.0:7677",
		WSAddr:          "0.0.0.0:2122",
		HTTPAddr:        "0.0.0.0:1516",
		SlotCount:       256,
		SegmentMaxBytes: 1024 * 1024 * 1024,
		CMDSendTimeout: duration{
			time.Second * 10,
		},
		Mode:            DebugMode,
		MessagePoolSize: 102400,
		EventPoolSize:   10240,
		Proto:           lmproto.New(),
		TimingWheelTick: duration{
			Duration: time.Millisecond * 10,
		},
		TimingWheelSize:     100,
		MaxMessagePerSecond: 100000,
		DeliveryMsgPoolSize: 100000,
		TmpChannelSuffix:    "@tmp",
		TimeoutScanInterval: duration{
			Duration: 1 * time.Second,
		},
		CreateIfChannelNotExist: false,
		ConversationSyncInterval: duration{
			Duration: time.Minute * 5,
		},
		ConversationSyncOnce:       100,
		ConversationOfUserMaxCount: 100,
		MsgTimeout: duration{
			Duration: 60 * time.Second,
		},
		MessageNotifyMaxCount: 20,
		MessageNotifyScanInterval: duration{
			Duration: time.Millisecond * 500,
		},
		NodeRPCMsgTimeout: duration{
			Duration: 20 * time.Second,
		},
		NodeRPCTimeoutScanInterval: duration{
			Duration: 1 * time.Second,
		},
	}

	return opts
}

// Load 从配置文件加载配置
func (o *Options) Load(fpath ...string) {
	configor.Load(o, fpath...)
	if strings.TrimSpace(o.DataDir) == "" {
		o.DataDir = fmt.Sprintf("limaodata-%d", o.NodeID)
	}

	if strings.TrimSpace(o.NodeRaftAddr) == "" {
		o.NodeRaftAddr = fmt.Sprintf("%s:6000", getIntranetIP())
	}
	if strings.TrimSpace(o.NodeAPIAddr) == "" {
		addrPairs := strings.Split(o.HTTPAddr, ":")
		portInt64, _ := strconv.ParseInt(addrPairs[1], 10, 64)
		o.NodeAPIAddr = fmt.Sprintf("http://%s:%d", getIntranetIP(), portInt64)
	}
	if strings.TrimSpace(o.NodeRPCAddr) == "" {
		o.NodeRPCAddr = fmt.Sprintf("%s:6001", getIntranetIP())
	}
	if strings.TrimSpace(o.NodeTCPAddr) == "" {
		addrPairs := strings.Split(o.Addr, ":")
		portInt64, _ := strconv.ParseInt(addrPairs[len(addrPairs)-1], 10, 64)
		externalIP, err := util.GetExternalIP()
		if err != nil {
			panic(err)
		}
		o.NodeTCPAddr = fmt.Sprintf("%s:%d", externalIP, portInt64)
	}
	if strings.TrimSpace(o.Mode) == TestMode {
		o.ConversationOfUserMaxCount = 100000
	}
	if strings.TrimSpace(o.Proxy) != "" {
		o.IsCluster = true
	}
	o.SlotBackupDir = path.Join(o.DataDir, "slotbackup")
	if strings.TrimSpace(o.SlotBackupDir) != "" {
		err := os.MkdirAll(o.SlotBackupDir, 0755)
		if err != nil {
			panic(err)
		}
	}
}

func getIntranetIP() string {
	intranetIPs, err := util.GetIntranetIP()
	if err != nil {
		panic(err)
	}
	if len(intranetIPs) > 0 {
		return intranetIPs[0]
	}
	return ""
}

// WebhookOn WebhookOn
func (o *Options) WebhookOn() bool {
	return strings.TrimSpace(o.Webhook) != ""
}

// HasDatasource 是否有配置数据源
func (o *Options) HasDatasource() bool {
	return strings.TrimSpace(o.Datasource) != ""
}

// IsTmpChannel 是否是临时频道
func (o *Options) IsTmpChannel(channelID string) bool {
	return strings.HasSuffix(channelID, o.TmpChannelSuffix)
}

// IsFakeChannel 是fake频道
func (o *Options) IsFakeChannel(channelID string) bool {
	return strings.Contains(channelID, "@")
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}
