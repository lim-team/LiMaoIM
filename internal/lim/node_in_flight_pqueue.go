package lim

import (
	"sync"
	"time"

	"github.com/lim-team/LiMaoIM/internal/db"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"github.com/pkg/errors"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"go.uber.org/zap"
)

// NodeInFlightData NodeInFlightData
type NodeInFlightData struct {
	db.NodeInFlightDataModel
	pri   int64 // 优先级的时间点 值越小越优先
	index int
}

// NodeInFlightQueue 正在投递的节点消息的队列
type NodeInFlightQueue struct {
	l             *LiMao
	infFlights    sync.Map
	inFlightPQ    nodeInFlightDataPqueue
	inFlightData  map[string]*NodeInFlightData
	inFlightMutex sync.Mutex
	limlog.Log
}

// NewNodeInFlightQueue NewNodeInFlightQueue
func NewNodeInFlightQueue(l *LiMao) *NodeInFlightQueue {
	return &NodeInFlightQueue{
		l:            l,
		infFlights:   sync.Map{},
		Log:          limlog.NewLIMLog("NodeInFlightQueue"),
		inFlightData: map[string]*NodeInFlightData{},
		inFlightPQ:   newNodeInFlightDataPqueue(1024),
	}
}

// startInFlightTimeout startInFlightTimeout
func (n *NodeInFlightQueue) startInFlightTimeout(data *NodeInFlightData) {
	now := time.Now()
	data.pri = now.Add(n.l.opts.NodeRPCMsgTimeout.Duration).UnixNano()
	n.pushInFlightMessage(data)
	n.addToInFlightPQ(data)
}

func (n *NodeInFlightQueue) addToInFlightPQ(data *NodeInFlightData) {
	n.inFlightMutex.Lock()
	defer n.inFlightMutex.Unlock()
	n.inFlightPQ.Push(data)

}
func (n *NodeInFlightQueue) pushInFlightMessage(data *NodeInFlightData) {
	n.inFlightMutex.Lock()
	defer n.inFlightMutex.Unlock()
	_, ok := n.inFlightData[data.No]
	if ok {
		return
	}
	n.inFlightData[data.No] = data

}

func (n *NodeInFlightQueue) finishMessage(no string) error {
	data, err := n.popInFlightMessage(no)
	if err != nil {
		return err
	}
	n.removeFromInFlightPQ(data)
	return nil
}
func (n *NodeInFlightQueue) removeFromInFlightPQ(data *NodeInFlightData) {
	n.inFlightMutex.Lock()
	if data.index == -1 {
		// this item has already been popped off the pqueue
		n.inFlightMutex.Unlock()
		return
	}
	n.inFlightPQ.Remove(data.index)
	n.inFlightMutex.Unlock()
}
func (n *NodeInFlightQueue) processInFlightQueue(t int64) {
	for {
		n.inFlightMutex.Lock()
		data, _ := n.inFlightPQ.PeekAndShift(t)
		n.inFlightMutex.Unlock()
		if data == nil {
			break
		}
		_, err := n.popInFlightMessage(data.No)
		if err != nil {
			break
		}
		// 开始投递
		n.l.startDeliveryNodeData(data)
	}
}

func (n *NodeInFlightQueue) popInFlightMessage(no string) (*NodeInFlightData, error) {
	n.inFlightMutex.Lock()
	defer n.inFlightMutex.Unlock()
	msg, ok := n.inFlightData[no]
	if !ok {
		return nil, errors.New("ID not in flight")
	}
	delete(n.inFlightData, no)
	return msg, nil
}

// Start 开始运行重试
func (n *NodeInFlightQueue) Start() {
	nodeInFlightDatas, err := n.l.store.GetNodeInFlightData() // TODO: 分布式情况多节点下，这里存在重复投递的可能，但是就算重复投递，客户端有去重机制所以也不影响，后面可以修正
	if err != nil {
		panic(err)
	}
	err = n.l.store.ClearNodeInFlightData()
	if err != nil {
		panic(err)
	}
	if len(nodeInFlightDatas) > 0 {
		for _, nodeInFlightDataModel := range nodeInFlightDatas {
			n.startInFlightTimeout(&NodeInFlightData{
				NodeInFlightDataModel: *nodeInFlightDataModel,
			})
		}

	}
	n.l.Schedule(n.l.opts.NodeRPCTimeoutScanInterval.Duration, func() {
		now := time.Now().UnixNano()
		n.processInFlightQueue(now)
	})
}

// Stop Stop
func (n *NodeInFlightQueue) Stop() {
	datas := make([]*db.NodeInFlightDataModel, 0)
	n.infFlights.Range(func(key, value interface{}) bool {
		datas = append(datas, &value.(*NodeInFlightData).NodeInFlightDataModel)
		return true
	})
	if len(datas) > 0 {
		n.Warn("存在节点投递数据未投递。", zap.Int("count", len(datas)))
		err := n.l.store.AddNodeInFlightData(datas)
		if err != nil {
			n.Error("异常退出", zap.Error(err), zap.String("data", util.ToJSON(datas)))
			return
		}
	}
	n.Info("正常退出")
}

type nodeInFlightDataPqueue []*NodeInFlightData

func newNodeInFlightDataPqueue(capacity int) nodeInFlightDataPqueue {
	return make(nodeInFlightDataPqueue, 0, capacity)
}

func (pq nodeInFlightDataPqueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *nodeInFlightDataPqueue) Push(x *NodeInFlightData) {
	n := len(*pq)
	c := cap(*pq)
	if n+1 > c {
		npq := make(nodeInFlightDataPqueue, n, c*2)
		copy(npq, *pq)
		*pq = npq
	}
	*pq = (*pq)[0 : n+1]
	x.index = n
	(*pq)[n] = x
	pq.up(n)
}

func (pq *nodeInFlightDataPqueue) Pop() *NodeInFlightData {
	n := len(*pq)
	c := cap(*pq)
	pq.Swap(0, n-1)
	pq.down(0, n-1)
	if n < (c/2) && c > 25 {
		npq := make(nodeInFlightDataPqueue, n, c/2)
		copy(npq, *pq)
		*pq = npq
	}
	x := (*pq)[n-1]
	x.index = -1
	*pq = (*pq)[0 : n-1]
	return x
}

func (pq *nodeInFlightDataPqueue) Remove(i int) *NodeInFlightData {
	n := len(*pq)
	if n-1 != i {
		pq.Swap(i, n-1)
		pq.down(i, n-1)
		pq.up(i)
	}
	x := (*pq)[n-1]
	x.index = -1
	*pq = (*pq)[0 : n-1]
	return x
}

func (pq *nodeInFlightDataPqueue) PeekAndShift(max int64) (*NodeInFlightData, int64) {
	if len(*pq) == 0 {
		return nil, 0
	}

	x := (*pq)[0]
	if x.pri > max {
		return nil, x.pri - max
	}
	pq.Pop()

	return x, 0
}

func (pq *nodeInFlightDataPqueue) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || (*pq)[j].pri >= (*pq)[i].pri {
			break
		}
		pq.Swap(i, j)
		j = i
	}
}

func (pq *nodeInFlightDataPqueue) down(i, n int) {
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && (*pq)[j1].pri >= (*pq)[j2].pri {
			j = j2 // = 2*i + 2  // right child
		}
		if (*pq)[j].pri >= (*pq)[i].pri {
			break
		}
		pq.Swap(i, j)
		i = j
	}
}
