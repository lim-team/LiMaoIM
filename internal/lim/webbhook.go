package lim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Event Event
type Event struct {
	Event string      `json:"event"` // 事件标示
	Data  interface{} `json:"data"`  // 事件数据
}

func (e *Event) String() string {
	return fmt.Sprintf("Event:%s Data:%v", e.Event, e.Data)
}

//TriggerEvent 触发事件
func (l *LiMao) TriggerEvent(event *Event) {
	if strings.TrimSpace(l.opts.Webhook) == "" { // 没设置webhook直接忽略
		return
	}
	err := l.eventPool.Submit(func() {
		jsonData, err := json.Marshal(event.Data)
		if err != nil {
			l.Error("webhook的event数据不能json化！", zap.Error(err))
			return
		}
		eventURL := fmt.Sprintf("%s?event=%s", l.opts.Webhook, event.Event)
		startTime := time.Now().UnixNano() / 1000 / 1000
		l.Debug("开始请求", zap.String("eventURL", eventURL))
		resp, err := http.Post(eventURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			l.Warn("调用webhook失败！", zap.String("webhook", l.opts.Webhook), zap.Error(err))
			return
		}
		l.Debug("耗时", zap.Int64("mill", time.Now().UnixNano()/1000/1000-startTime))
		if resp.StatusCode != 200 {
			l.Warn("调用webhook失败！", zap.Int("status", resp.StatusCode), zap.String("webhook", l.opts.Webhook))
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			l.Warn("读取body数据失败！", zap.String("webhook", l.opts.Webhook))
			return
		}
		l.Debug("webhook返回数据", zap.String("body", string(body)))
	})
	if err != nil {
		l.Error("提交事件失败", zap.Error(err))
	}
}

// 通知上层应用 TODO: 此初报错可以做一个邮件报警处理类的东西，
func (l *LiMao) notifyQueueLoop() {
	errorSleepTime := time.Second * 1 // 发生错误后sleep时间
	ticker := time.NewTicker(l.opts.MessageNotifyScanInterval.Duration)
	if l.opts.WebhookOn() {
		for {
			if !l.opts.IsCluster || !l.clusterManager.IsLeader() { // 只有主节点才进行推送消息到业务服务（TODO：这里可优化成从节点去推送，这样可以减轻主节点的压力）
				time.Sleep(time.Second * 5)
				continue
			}
			messages, err := l.store.GetMessagesOfNotifyQueue(l.opts.MessageNotifyMaxCount)
			if err != nil {
				l.Error("获取通知队列内的消息失败！", zap.Error(err))
				time.Sleep(errorSleepTime) // 如果报错就休息下
				continue
			}
			if len(messages) > 0 {
				messageResps := make([]*MessageResp, 0, len(messages))
				for _, msg := range messages {
					resp := &MessageResp{}
					resp.from(msg)
					messageResps = append(messageResps, resp)
				}
				messageData, err := json.Marshal(messageResps)
				if err != nil {
					l.Error("第三方消息通知的event数据不能json化！", zap.Error(err))
					time.Sleep(errorSleepTime) // 如果报错就休息下
					continue
				}
				resp, err := http.Post(fmt.Sprintf("%s?event=%s", l.opts.Webhook, EventMsgNotify), "application/json", bytes.NewBuffer(messageData))
				if err != nil {
					l.Warn("调用第三方消息通知失败！", zap.String("Webhook", l.opts.Webhook), zap.Error(err))
					time.Sleep(errorSleepTime) // 如果报错就休息下
					continue
				}
				if resp.StatusCode != 200 {
					l.Warn("第三方消息通知接口返回状态错误！", zap.Int("status", resp.StatusCode), zap.String("Webhook", l.opts.Webhook))
					time.Sleep(errorSleepTime) // 如果报错就休息下
					continue
				}

				messageIDs := make([]int64, 0, len(messages))
				for _, message := range messages {
					messageIDs = append(messageIDs, message.MessageID)
				}
				err = l.store.RemoveMessagesOfNotifyQueue(messageIDs)
				if err != nil {
					l.Warn("从通知队列里移除消息失败！", zap.Error(err), zap.Int64s("messageIDs", messageIDs), zap.String("Webhook", l.opts.Webhook))
					time.Sleep(errorSleepTime) // 如果报错就休息下
					continue
				}
			}

			select {
			case <-ticker.C:
			case <-l.stoped:
				return
			}
		}
	}
}
