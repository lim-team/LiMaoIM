package lim

import (
	"log"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tangtaoit/go-metrics"
	"github.com/tangtaoit/limnet/pkg/limutil/sync/atomic"
)

// Monitor 监控
type Monitor struct {
	namespace string
	// ----- 连接数统计 -----
	connCountGauge metrics.Gauge
	connCount      atomic.Int64
	// ----- 上行  -----
	upstreamTotalTrafficGauge metrics.Gauge // 上行总流量统计
	upstreamTotalTrafficCount atomic.Int64
	upstreamTrafficRate       metrics.Meter // 上行流量数率
	upstreamTotalPacketGauge  metrics.Gauge // 上行总包
	upstreamTotalPacketCount  atomic.Int64
	upstreamPacketRate        metrics.Meter // 上行包速率
	// ----- 下行  -----
	downstreamTotalTrafficGauge metrics.Gauge // 下行总流量统计
	downstreamTotalTrafficCount atomic.Int64
	downstreamTrafficRate       metrics.Meter // 上行流量数率
	downstreamTotalPacketGauge  metrics.Gauge // 上行总包
	downstreamTotalPacketCount  atomic.Int64
	downstreamPacketRate        metrics.Meter // 上行包速率

	// top 统计（统计top N的用户的上行下行数据）

	// 客户端包
	clientPacketGaugeVec *prometheus.GaugeVec
	clientDataGaugeVec   *prometheus.GaugeVec

	l        *LiMao
	registry metrics.Registry
}

// NewMonitor 创建监控对象
func NewMonitor(l *LiMao) *Monitor {

	m := &Monitor{
		l:              l,
		registry:       metrics.NewRegistry(),
		connCountGauge: metrics.NewGauge(),
		namespace:      "limao",
		// --- 上行 ----
		upstreamTotalTrafficGauge: metrics.NewGauge(),
		upstreamTrafficRate:       metrics.NewMeter(),
		upstreamTotalPacketGauge:  metrics.NewGauge(),
		upstreamPacketRate:        metrics.NewMeter(),
		// --- 下行 ----
		downstreamTotalTrafficGauge: metrics.NewGauge(),
		downstreamTrafficRate:       metrics.NewMeter(),
		downstreamTotalPacketGauge:  metrics.NewGauge(),
		downstreamPacketRate:        metrics.NewMeter(),
	}

	m.registry.Register("连接数", m.connCountGauge)
	m.registry.Register("上行流量", m.upstreamTotalTrafficGauge)
	m.registry.Register("上行包数", m.upstreamTotalPacketGauge)
	m.registry.Register("上行流量速率", m.upstreamTrafficRate)
	m.registry.Register("上行包速率", m.upstreamPacketRate)

	m.registry.Register("下行流量", m.downstreamTotalTrafficGauge)
	m.registry.Register("下行包数", m.downstreamTotalPacketGauge)
	m.registry.Register("下行流量速率", m.downstreamTrafficRate)
	m.registry.Register("下行包速率", m.downstreamPacketRate)

	if l.opts.Mode == TestMode {
		go m.print(time.Second * 20)
	} else {
		go m.print(time.Minute * 15)
	}

	return m
}

func (m *Monitor) print(freq time.Duration) {

	ch := make(chan interface{})
	go func(channel chan interface{}) {
		for _ = range time.Tick(freq) {
			channel <- struct{}{}
		}
	}(ch)
	m.logScaledOnCue(ch)
}

func (m *Monitor) logScaledOnCue(ch chan interface{}) {
	l := log.New(os.Stdout, "metrics: ", log.Lmicroseconds)
	for _ = range ch {
		l.Printf("--------------------监控打印--------------------\n")
		m.registry.Each(func(name string, i interface{}) {
			switch i {
			case m.upstreamTotalTrafficGauge, m.downstreamTotalTrafficGauge:
				l.Printf("%s: %0.4f m\n", name, float64(i.(metrics.Gauge).Value())/1024.0/1024.0)
				break
			case m.upstreamTotalPacketGauge, m.downstreamTotalPacketGauge, m.connCountGauge:
				l.Printf("%s: %d \n", name, i.(metrics.Gauge).Value())
				break
			case m.upstreamTrafficRate, m.downstreamTrafficRate:
				meter := i.(metrics.Meter)
				l.Printf("%s \n", name)
				l.Printf("  1分钟内的速率 %0.4f m/s \n", meter.Rate1()/1024.0/1024.0)
				l.Printf("  5分钟内的速率 %0.4f m/s \n", meter.Rate5()/1024.0/1024.0)
				l.Printf("  15分钟内的速率 %0.4f m/s \n", meter.Rate15()/1024.0/1024.0)
				l.Printf("  平均速率 %0.4f m/s", meter.RateMean()/1024.0/1024.0)
				break
			case m.upstreamPacketRate, m.downstreamPacketRate:
				meter := i.(metrics.Meter)
				l.Printf("%s \n", name)
				l.Printf("  1分钟内的速率 %0.0f个/s \n", meter.Rate1())
				l.Printf("  5分钟内的速率 %0.0f个/S \n", meter.Rate5())
				l.Printf("  15分钟内的速率 %0.0f个/s \n", meter.Rate15())
				l.Printf("  平均速率 %0.0f个/s", meter.RateMean())
				break

			}
		})
	}
}

// Start 开始
func (m *Monitor) Start() {
	// 监听数据流量
	//m.monitorTrafficTotal()
	// 监控包流量
	//m.monitorPacketTotal()
}

// ConnInc 连接递增
func (m *Monitor) ConnInc() {
	m.connCount.Add(1)
	m.connCountGauge.Update(m.connCount.Get())

}

// ConnDec 连接递减
func (m *Monitor) ConnDec() {
	m.connCount.Add(-1)
	m.connCountGauge.Update(m.connCount.Get())
}

// UpstreamAdd 上行流量增加
func (m *Monitor) UpstreamAdd(count int) {
	m.upstreamTotalTrafficCount.Add(count)
	m.upstreamTotalTrafficGauge.Update(m.upstreamTotalTrafficCount.Get())

	m.upstreamTrafficRate.Mark(int64(count))

}

// DownstreamAdd 上行流量增加
func (m *Monitor) DownstreamAdd(count int) {
	m.downstreamTotalTrafficCount.Add(count)
	m.downstreamTotalTrafficGauge.Update(m.downstreamTotalTrafficCount.Get())

	m.downstreamTrafficRate.Mark(int64(count))

}

// UpstreamPacketInc 上行包递增
func (m *Monitor) UpstreamPacketInc() {
	m.upstreamTotalPacketCount.Add(1)
	m.upstreamTotalPacketGauge.Update(m.upstreamTotalPacketCount.Get())

	m.upstreamPacketRate.Mark(1)
}

// DownstreamPacketInc 上行包递增
func (m *Monitor) DownstreamPacketInc() {
	m.downstreamTotalPacketCount.Add(1)
	m.downstreamTotalPacketGauge.Update(m.downstreamTotalPacketCount.Get())

	m.downstreamPacketRate.Mark(1)
}

// // PacketClientSendSet 设置客户端发送包数量
// func (m *Monitor) PacketClientSendSet(id int64, count int64) {
// 	m.clientPacketGaugeVec.WithLabelValues(fmt.Sprintf("%d_send", id)).Set(float64(count))
// }

// // PacketClientRecvSet 设置客户端接受包数量
// func (m *Monitor) PacketClientRecvSet(id int64, count int64) {
// 	m.clientPacketGaugeVec.WithLabelValues(fmt.Sprintf("%d_recv", id)).Set(float64(count))
// }

// // DataClientSendSet 设置客户端发送数据大小
// func (m *Monitor) DataClientSendSet(id int64, count int64) {
// 	m.clientDataGaugeVec.WithLabelValues(fmt.Sprintf("%d_send", id)).Set(float64(count))
// }

// DataClientRecvSet 设置客户端接受数据大小
// func (m *Monitor) DataClientRecvSet(id int64, count int64) {
// 	m.clientDataGaugeVec.WithLabelValues(fmt.Sprintf("%d_recv", id)).Set(float64(count))
// }
