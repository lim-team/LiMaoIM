package lim

import (
	"fmt"
	"hash/crc32"
	"strings"
	"time"

	"github.com/lim-team/LiMaoIM/pkg/limlog"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
	"go.uber.org/zap"
)

// GetSlotNum GetSlotNum
func GetSlotNum(slotCount uint, channelID string, channelType uint8) uint {
	value := crc32.ChecksumIEEE([]byte(fmt.Sprintf("%s-%d", channelID, channelType)))
	return uint(value % uint32(slotCount))
}

// State State
type State int32

const (
	// StateWaitCluster StateWaitCluster
	StateWaitCluster State = iota
	// StateClustered StateClustered
	StateClustered
)

// ConnStatus 连接状态
type ConnStatus int

const (
	// ConnStatusNoAuth 未认证
	ConnStatusNoAuth ConnStatus = iota
	// ConnStatusAuthed 已认证
	ConnStatusAuthed
)

// Int Int
func (c ConnStatus) Int() int {
	return int(c)
}

const (
	// ChannelTypePerson 个人频道
	ChannelTypePerson uint8 = 1
	// ChannelTypeGroup 群频道
	ChannelTypeGroup uint8 = 2 // 群组频道
)

type everyScheduler struct {
	Interval time.Duration
}

func (s *everyScheduler) Next(prev time.Time) time.Time {
	return prev.Add(s.Interval)
}

// GetFakeChannelID 通过接受包获取伪频道ID
func GetFakeChannelID(recvPacket *lmproto.RecvPacket) string {
	if recvPacket.ChannelType != ChannelTypePerson {
		return recvPacket.ChannelID
	}
	return GetFakeChannelIDWith(recvPacket.FromUID, recvPacket.ChannelID)
}

// GetFakeChannelIDWith GetFakeChannelIDWith
func GetFakeChannelIDWith(fromUID, toUID string) string {
	// TODO：这里可能会出现相等的情况 ，如果相等可以截取一部分再做hash直到不相等，后续完善
	fromUIDHash := util.HashCrc32(fromUID)
	toUIDHash := util.HashCrc32(toUID)
	if fromUIDHash > toUIDHash {
		return fmt.Sprintf("%s@%s", fromUID, toUID)
	}
	if fromUIDHash == toUIDHash {
		limlog.Warn("生成的fromUID的Hash和toUID的Hash是相同的！！", zap.Uint32("fromUIDHash", fromUIDHash), zap.Uint32("toUIDHash", toUIDHash), zap.String("fromUID", fromUID), zap.String("toUID", toUID))
	}

	return fmt.Sprintf("%s@%s", toUID, fromUID)
}

// GetOtherSideUIDByFakeChannelID 通过伪频道ID获取聊天对方UID uid: 当前用户
func GetOtherSideUIDByFakeChannelID(uid string, fakeChannelID string) string {
	uids := strings.Split(fakeChannelID, "@")
	if len(uids) == 2 {
		if uid == uids[0] {
			return uids[1]
		}
		return uids[0]
	}
	return ""
}

const (
	// EventMsgNotify 消息通知（将所有消息通知到第三方程序）
	EventMsgNotify = "msg.notify"
	// EventOnlineStatus 用户在线状态
	EventOnlineStatus = "user.onlinestatus"
)
