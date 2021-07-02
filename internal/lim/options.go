package lim

import (
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/configor"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
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
	Port                       int              `env:"port"`
	Mode                       string           `env:"mode"` // mode
	DataDir                    string           `env:"dataDir"`
	SlotCount                  int              `env:"slotCount"` // slot count
	SegmentMaxBytes            int64            `env:"segmentMaxBytes"`
	CMDSendTimeout             duration         `env:"cmdSendTimeout"`
	MessagePoolSize            int              `env:"messagePoolSize"` // The size of the coroutine pool for processing packets
	DeliveryMsgPoolSize        int              `env:"deliveryMsgPoolSize"`
	MsgTimeout                 duration         `env:"msgTimeout"`          // Message sending timeout time, after this time it will try again
	TimeoutScanInterval        duration         `env:"timeoutScanInterval"` //  Message timeout queue scan frequency
	Proto                      lmproto.Protocol // protocol
	Webhook                    string           `env:"Webhook"`                   // The URL address of the third party that receives the message notification
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
}

// NewOptions NewOptions
func NewOptions() *Options {

	opts := &Options{
		NodeID:          1,
		Addr:            "tcp://0.0.0.0:7677",
		WSAddr:          "0.0.0.0:2122",
		HTTPAddr:        "0.0.0.0:1516",
		SlotCount:       256,
		SegmentMaxBytes: 1024 * 1024 * 1024,
		CMDSendTimeout: duration{
			time.Second * 10,
		},
		Mode:            ReleaseMode,
		MessagePoolSize: 102400,
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
	}

	return opts
}

// Load 从配置文件加载配置
func (o *Options) Load(path ...string) {
	configor.Load(o, path...)
	if strings.TrimSpace(o.DataDir) == "" {
		o.DataDir = fmt.Sprintf("limaodata-%d", o.NodeID)
	}
	if o.Mode == TestMode {
		o.ConversationOfUserMaxCount = 100000
	}
}

// WebhookOn WebhookOn
func (o *Options) WebhookOn() bool {
	return strings.TrimSpace(o.Webhook) != ""
}

// HasDatasource 是否有配置数据源
func (o *Options) HasDatasource() bool {
	return strings.TrimSpace(o.Datasource) != ""
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}
