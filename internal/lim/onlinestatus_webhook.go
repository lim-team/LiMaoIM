package lim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lim-team/LiMaoIM/pkg/limlog"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"go.uber.org/zap"
)

// OnlineStatusWebhook 在线状态webhook
type OnlineStatusWebhook struct {
	statusLock sync.RWMutex
	statusList []string
	l          *LiMao
	limlog.Log
}

// NewOnlineStatusWebhook NewOnlineStatusWebhook
func NewOnlineStatusWebhook(l *LiMao) *OnlineStatusWebhook {
	w := &OnlineStatusWebhook{
		statusList: make([]string, 0),
		l:          l,
		Log:        limlog.NewLIMLog("OnlineStatusWebhook"),
	}
	go w.loopNotify()
	return w
}

// Online 用户在线
func (w *OnlineStatusWebhook) Online(uid string, deviceFlag lmproto.DeviceFlag) {
	w.statusLock.Lock()
	defer w.statusLock.Unlock()
	w.statusList = append(w.statusList, fmt.Sprintf("%s-%d-%d", uid, deviceFlag, 1))

	w.Debug("用户上线", zap.String("uid", uid), zap.Uint8("deviceFlag", deviceFlag.ToUint8()))
}

// Offline 用户离线
func (w *OnlineStatusWebhook) Offline(uid string, deviceFlag lmproto.DeviceFlag) {
	w.statusLock.Lock()
	defer w.statusLock.Unlock()
	w.statusList = append(w.statusList, fmt.Sprintf("%s-%d-%d", uid, deviceFlag, 0))

	w.Debug("用户下线", zap.String("uid", uid), zap.Uint8("deviceFlag", deviceFlag.ToUint8()))
}

func (w *OnlineStatusWebhook) loopNotify() {
	if strings.TrimSpace(w.l.opts.Webhook) == "" {
		return
	}
	opLen := 0 // 最后一次操作在线状态数组的长度
	for {
		if opLen == 0 {
			opLen = len(w.statusList)
		}
		if opLen == 0 {
			time.Sleep(time.Second * 2) // 没有数据就休息2秒
			continue
		}
		jsonData, err := json.Marshal(w.statusList[:opLen])
		if err != nil {
			w.Error("webhook的event数据不能json化！", zap.Error(err))
			time.Sleep(time.Second * 1)
			continue
		}
		eventURL := fmt.Sprintf("%s?event=%s", w.l.opts.Webhook, EventOnlineStatus)
		startTime := time.Now().UnixNano() / 1000 / 1000
		w.Debug("开始请求", zap.String("eventURL", eventURL))
		w.Debug("请求参数", zap.String("body", string(jsonData)))
		resp, err := http.Post(eventURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			w.Warn("调用webhook失败！", zap.String("eventURL", eventURL), zap.Error(err))
			time.Sleep(time.Second * 1)
			continue
		}
		w.Debug("耗时", zap.Int64("mill", time.Now().UnixNano()/1000/1000-startTime))
		if resp.StatusCode != 200 {
			w.Warn("调用webhook失败！", zap.Int("status", resp.StatusCode), zap.String("eventURL", eventURL))
			time.Sleep(time.Second * 1)
			continue
		}
		w.statusLock.Lock()
		w.statusList = w.statusList[opLen:]
		opLen = 0
		w.statusLock.Unlock()

	}
}
