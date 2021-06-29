package lim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// 通知上层应用 TODO: 此初报错可以做一个邮件报警处理类的东西，
func (l *LiMao) notifyQueueLoop() {
	errorSleepTime := time.Second * 1 // 发生错误后sleep时间
	ticker := time.NewTicker(l.opts.MessageNotifyScanInterval.Duration)
	if l.opts.WebhookOn() {
		for {
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
