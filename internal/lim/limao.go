package lim

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lim-team/LiMaoIM/internal/lim/rpc"

	"github.com/RussellLuo/timingwheel"
	"github.com/lim-team/LiMaoIM/pkg/limlog"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/panjf2000/ants/v2"
	"github.com/tangtaoit/limnet"
	linetlimglog "github.com/tangtaoit/limnet/pkg/limlog"
	"github.com/tangtaoit/limnet/pkg/limutil"
	"github.com/tangtaoit/limnet/pkg/pool/goroutine"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

var (
	// ErrorDoCommandTimeout  Execution command timed out
	ErrorDoCommandTimeout = errors.New("do command timeout")
)

// NewCluster NewCluster
var NewCluster func(l *LiMao) IClusterManager // 创建一个分布式实现
// LiMao core
type LiMao struct {
	opts *Options
	lnet *limnet.LIMNet // lim net
	limlog.Log
	clientManager       *ClientManager // client manager
	stoped              chan struct{}
	done                chan struct{}
	startCompleteC      chan struct{}
	messagePool         *goroutine.Pool          // message goroutine pool
	deliveryMsgPool     *goroutine.Pool          // Message delivery goroutine pool
	eventPool           *goroutine.Pool          // 事件传递线程池
	timingWheel         *timingwheel.TimingWheel // Time wheel delay task
	packetHandler       *PacketHandler           // Logic processor
	waitGroupWrapper    *limutil.WaitGroupWrapper
	monitor             *Monitor         // Data monitoring
	clusterManager      IClusterManager  // cluster management
	protocol            lmproto.Protocol //Protocol interface
	messageRate         *rate.Limiter    // Message rate
	store               Storage
	channelManager      *ChannelManager
	nodeRPCServer       *rpc.Server       // 节点rpc服务，与节点通讯用
	apiServer           *APIServer        // api服务
	retryQueue          *RetryQueue       // 重试队列
	systemUIDManager    *SystemUIDManager // System uid management, system uid can send messages to everyone without any restrictions
	conversationManager *ConversationManager
	datasource          IDatasource // 数据源（提供数据源 订阅者，黑名单，白名单这些数据可以交由第三方提供）
	onlineStatusWebhook *OnlineStatusWebhook
	nodeInFlightQueue   *NodeInFlightQueue // 正在往节点投递的节点消息
	nodeRemoteCall      *NodeRemoteCall

	DoCommand         func(cmd *CMD) error // 执行命令（分布式副本之间执行命令）
	startCompleteOnce sync.Once
}

// New New
func New(opts *Options) *LiMao {
	l := &LiMao{
		opts:             opts,
		Log:              limlog.NewLIMLog("LiMao"),
		done:             make(chan struct{}),
		stoped:           make(chan struct{}),
		waitGroupWrapper: limutil.NewWaitGroupWrapper("limao"),
		protocol:         opts.Proto,
		clientManager:    NewClientManager(),
		timingWheel:      timingwheel.NewTimingWheel(opts.TimingWheelTick.Duration, opts.TimingWheelSize),
		messageRate:      rate.NewLimiter(rate.Limit(opts.MaxMessagePerSecond), opts.MaxMessagePerSecond),
	}
	err := os.MkdirAll(l.opts.DataDir, 0755)
	if err != nil {
		panic(err)
	}
	l.store = NewStorage(l)
	l.nodeInFlightQueue = NewNodeInFlightQueue(l)
	l.channelManager = NewChannelManager(l)
	if l.opts.IsCluster && NewCluster != nil {
		l.clusterManager = NewCluster(l)
	}
	l.packetHandler = NewPacketHandler(l)
	l.onlineStatusWebhook = NewOnlineStatusWebhook(l)
	l.systemUIDManager = NewSystemUIDManager(l)
	l.retryQueue = NewRetryQueue(l)
	l.apiServer = NewAPIServer(l)
	l.monitor = NewMonitor(l)
	l.datasource = NewDatasource(l)
	l.lnet = limnet.New(l, limnet.WithAddr(l.opts.Addr), limnet.WithWSAddr(l.opts.WSAddr), limnet.WithUnPacket(limUnpacket))
	l.conversationManager = NewConversationManager(l)
	if opts.Mode == TestMode {
		linetlimglog.TestMode = true
		limlog.TestMode = true
	} else if opts.Mode == DebugMode {
		gin.SetMode(gin.DebugMode)
		limlog.SetLevel(zap.DebugLevel)
		linetlimglog.SetLevel(zap.DebugLevel)
	} else {
		gin.SetMode(gin.ReleaseMode)
		limlog.SetLevel(zap.InfoLevel)
		linetlimglog.SetLevel(zap.InfoLevel)
	}
	l.nodeRemoteCall = NewNodeRemoteCall(l)

	options := ants.Options{ExpiryDuration: 10 * time.Second, Nonblocking: true}
	l.messagePool, err = ants.NewPool(l.opts.MessagePoolSize, ants.WithOptions(options))
	if err != nil {
		panic(err)
	}
	l.deliveryMsgPool, err = ants.NewPool(l.opts.DeliveryMsgPoolSize, ants.WithOptions(options), ants.WithPanicHandler(func(err interface{}) {
		fmt.Println("消息投递panic->", err)
	}))
	if err != nil {
		panic(err)
	}
	l.eventPool, err = ants.NewPool(l.opts.EventPoolSize, ants.WithOptions(options), ants.WithPanicHandler(func(err interface{}) {
		fmt.Println("事件池panic->", err)
	}))
	if err != nil {
		panic(err)
	}
	// 节点之间通讯服务
	l.nodeRPCServer = rpc.NewServer(l.protocol, l.packetHandler, l.opts.NodeRPCAddr)

	return l
}

// GetOptions GetOptions
func (l *LiMao) GetOptions() *Options {
	return l.opts
}

// GetStore GetStore
func (l *LiMao) GetStore() Storage {
	return l.store
}

// GetConversationManager GetConversationManager
func (l *LiMao) GetConversationManager() *ConversationManager {
	return l.conversationManager
}

// Start Start
func (l *LiMao) Start(startCompleteC ...chan struct{}) error {
	if len(startCompleteC) > 0 {
		l.startCompleteC = startCompleteC[0]
	}

	if l.opts.IsCluster {
		err := l.clusterManager.Start()
		if err != nil {
			return err
		}
		l.clusterManager.LeaderUpdated(func(leaderID uint32) {
			if leaderID != 0 {
				l.startCompleteOnce.Do(func() {
					l.ready()
					if l.startCompleteC != nil {
						go func() {
							l.startCompleteC <- struct{}{}
						}()
					}
				})
			}
		})
	} else {
		l.ready()
		if l.startCompleteC != nil {
			go func() {
				l.startCompleteC <- struct{}{}
			}()
		}

	}
	return nil
}

func (l *LiMao) ready() {
	l.nodeInFlightQueue.Start()
	// Message polling notification to a third party
	l.waitGroupWrapper.Wrap(func() {
		l.notifyQueueLoop()
	})
	l.timingWheel.Start()

	l.nodeRPCServer.Start()

	// conversation management is on
	l.conversationManager.Start()

	l.waitGroupWrapper.Wrap(func() {
		l.lnet.Run()
	})
	//Run API service
	l.apiServer.Start()
	l.print()

}

func (l *LiMao) print() {
	fmt.Println(`
	_ _                        
	| (_)                      
	| |_ _ __ ___   __ _  ___  
	| | | '_ ` + "`" + ` _ \ / _` + "`" + ` |/ _ \ 
	| | | | | | | | (_| | (_) |
	|_|_|_| |_| |_|\__,_|\___/ 
							  
							  
	`)
	if l.opts.Mode == TestMode {
		l.Info("已开启测试模式！测试模式仅供测试使用！")
	}
	l.Info("服务模式", zap.String("mode", l.opts.Mode))
	l.Info("Socket服务", zap.String("addr", l.opts.Addr))
	l.Info("WebSocket服务", zap.String("wsAddr", l.opts.WSAddr))
	l.Info("HTTP服务", zap.String("httpAddr", fmt.Sprintf("http://%s", l.opts.HTTPAddr)))
	l.Info("API文档地址", zap.String("apidocs", fmt.Sprintf("http://%s/api", l.opts.HTTPAddr)))
}

// Stop Stop
func (l *LiMao) Stop() error {
	// close(l.stoped)
	// <-l.done
	return l.onStop()
}

// Schedule 延迟任务
func (l *LiMao) Schedule(interval time.Duration, f func()) {
	l.timingWheel.ScheduleFunc(&everyScheduler{
		Interval: interval,
	}, f)
}

// -------------------- event handler --------------------

// OnConnect establish connection
func (l *LiMao) OnConnect(c limnet.Conn) {
	l.monitor.ConnInc()
}

// OnPacket Packet received
func (l *LiMao) OnPacket(c limnet.Conn, data []byte) (out []byte) {

	// Upstream traffic statistics
	l.monitor.UpstreamAdd(len(data))

	// 处理包
	offset := 0
	packets := make([]lmproto.Frame, 0)
	for len(data) > offset {
		packet, size, err := l.protocol.DecodePacket(data[offset:], c.Version())
		if err != nil { //
			l.Warn("Failed to decode the message", zap.Error(err))
			c.Close()
			return
		}
		packets = append(packets, packet)
		l.monitor.UpstreamPacketInc() // Increasing total package
		offset += size
		if c.Status() == ConnStatusNoAuth.Int() && packet.GetPacketType() != lmproto.CONNECT {
			l.Warn("The first package should be the connection package! The connection will be closed")
			c.Close()
			return
		}

	}
	for _, packet := range packets {
		l.handlePacket(c, packet)
	}

	return
}

func (l *LiMao) handlePacket(c limnet.Conn, packet lmproto.Frame) {
	switch packet.GetPacketType() {
	case lmproto.CONNECT: // connect
		l.messagePool.Submit(func() {
			l.packetHandler.handleConnect(c, packet.(*lmproto.ConnectPacket))
		})
	case lmproto.PING: // ping
		client := l.clientManager.Get(c.GetID())
		if client == nil {
			l.Warn("发送消息的客户端没有找到，不处理此条消息！", zap.Any("pingPacket", packet))
			return
		}
		l.packetHandler.handlePing(client)
	case lmproto.SEND: //  send
		client := l.clientManager.Get(c.GetID())
		if client == nil {
			l.Warn("发送消息的客户端没有找到，不处理此条消息！", zap.Any("sendPacket", packet))
			return
		}
		err := l.messageRate.Wait(context.Background()) // Rate control
		if err != nil {
			l.Warn("messageRate wait fail", zap.Error(err))
		}
		l.messagePool.Submit(func() {
			client.clientSendPacketInc() // 客户端发送包统计
			l.packetHandler.HandleSend(client, packet.(*lmproto.SendPacket))
		})

		break
	case lmproto.RECVACK:
		client := l.clientManager.Get(c.GetID())
		if client == nil {
			l.Warn("发送消息的客户端没有找到，不处理此条消息！", zap.Any("sendPacket", packet))
			return
		}
		l.packetHandler.handleRecvack(client, packet.(*lmproto.RecvackPacket))
		break
	}
}

// OnClose 连接关闭
func (l *LiMao) OnClose(c limnet.Conn) {
	l.monitor.ConnDec() // 连接数递减
	l.packetHandler.handleDisconnect(c)
}

func (l *LiMao) onStop() error {
	if l.lnet != nil {
		err := l.lnet.Stop()
		if err != nil {
			return err
		}
	}

	l.timingWheel.Stop()
	l.nodeInFlightQueue.Stop()
	// conversation management is off
	l.conversationManager.Stop()

	if l.opts.IsCluster {
		err := l.clusterManager.Stop()
		if err != nil {
			return err
		}
	} else {
		err := l.store.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *LiMao) handleLocalSubscribersMessage(m *Message, subscribers []string) error {
	subscriberSeqMap, err := l.storeMessageToUserQueueIfNeed(m, subscribers)
	if err != nil {
		return err
	}
	l.conversationManager.PushMessage(m, subscribers)
	l.startDeliveryMsg(m, subscriberSeqMap, subscribers...)
	return nil
}
