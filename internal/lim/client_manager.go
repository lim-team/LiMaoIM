package lim

import (
	"sync"

	"github.com/lim-team/LiMaoIM/pkg/lmproto"
)

// TODO: 需要优化

// ClientManager 客户端管理
type ClientManager struct {
	userClientMap sync.Map
	clientMap     sync.Map
	sync.RWMutex
}

// NewClientManager 创建一个默认的客户端管理者
func NewClientManager() *ClientManager {
	return &ClientManager{userClientMap: sync.Map{}, clientMap: sync.Map{}}
}

// Add 添加客户端
func (m *ClientManager) Add(client *Client) {
	m.clientMap.Store(client.GetID(), client)
	users, loaded := m.userClientMap.LoadOrStore(client.uid, []int64{client.GetID()})
	if loaded {
		m.Lock()
		defer m.Unlock()
		userList := users.([]int64)
		userList = append(userList, client.GetID())
		m.userClientMap.Store(client.uid, userList)

	}
}

// Get 获取客户端
func (m *ClientManager) Get(id int64) *Client {
	client, _ := m.clientMap.Load(id)
	if client != nil {
		return client.(*Client)
	}
	return nil
}

// Remove 移除客户端
func (m *ClientManager) Remove(id int64) {
	clientObj, _ := m.clientMap.Load(id)
	if clientObj == nil {
		return
	}
	client := clientObj.(*Client)
	m.clientMap.Delete(id)
	users, _ := m.userClientMap.Load(client.uid)
	if users != nil {
		m.Lock()
		defer m.Unlock()
		userList := users.([]int64)
		for index, id := range userList {
			if id == client.GetID() {
				userList = append(userList[:index], userList[index+1:]...)
				m.userClientMap.Store(client.uid, userList)
				break
			}
		}
	}
}

// GetClientsWithUID 通过用户uid获取客户端集合
func (m *ClientManager) GetClientsWithUID(uid string) []*Client {
	users, _ := m.userClientMap.Load(uid)
	if users == nil {
		return nil
	}

	// TODO: 这里存在同一个账号多个设备存在并发问题 加锁会影响总个IM性能，暂时观察
	userList := users.([]int64)
	// if len(userList) > 1 {
	// 	m.Lock()
	// 	defer m.Unlock()
	// }
	clients := make([]*Client, 0, len(userList))
	for _, id := range userList {
		client, _ := m.clientMap.Load(id)
		if client != nil {
			clients = append(clients, client.(*Client))
		}
	}
	return clients
}

// GetOnlineUIDs 传一批uids 返回在线的uids
func (m *ClientManager) GetOnlineUIDs(uids []string) []string {
	if len(uids) == 0 {
		return make([]string, 0)
	}
	var onlineUIDS = make([]string, 0, len(uids))
	for _, uid := range uids {
		if _, ok := m.userClientMap.Load(uid); ok {
			onlineUIDS = append(onlineUIDS, uid)
		}

	}
	return onlineUIDS
}

// GetClientWith 查询设备
func (m *ClientManager) GetClientWith(uid string, deviceFlag lmproto.DeviceFlag) *Client {
	clients := m.GetClientsWithUID(uid)
	if len(clients) == 0 {
		return nil
	}
	for _, client := range clients {
		if client.deviceFlag == deviceFlag {
			return client
		}
	}
	return nil
}
