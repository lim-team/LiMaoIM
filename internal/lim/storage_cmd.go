package lim

import (
	"fmt"

	"github.com/lim-team/LiMaoIM/internal/db"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/lim-team/LiMaoIM/pkg/util"
)

// CMDType CMDType
type CMDType int

const (
	// CMDUnknown unknown
	CMDUnknown CMDType = iota
	// CMDAppendMessage append message
	CMDAppendMessage
	// CMDAddOrUpdateChannel CMDAddOrUpdateChannel
	CMDAddOrUpdateChannel
	// CMDUpdateUserToken CMDUpdateUserToken
	CMDUpdateUserToken
	// CMDAddSubscribers CMDAddSubscribers
	CMDAddSubscribers
	// CMDRemoveAllSubscriber CMDRemoveAllSubscriber
	CMDRemoveAllSubscriber
	// CMDRemoveSubscribers CMDRemoveSubscribers
	CMDRemoveSubscribers
	// CMDDeleteChannel CMDDeleteChannel
	CMDDeleteChannel
	// CMDDeleteChannelAndClearMessages CMDDeleteChannelAndClearMessages
	CMDDeleteChannelAndClearMessages
	// CMDAppendMessageOfUser CMDAppendMessageOfUser
	CMDAppendMessageOfUser
	// CMDAppendMessageOfNotifyQueue CMDAppendMessageOfNotifyQueue
	CMDAppendMessageOfNotifyQueue
	// CMDRemoveMessagesOfNotifyQueue CMDRemoveMessagesOfNotifyQueue
	CMDRemoveMessagesOfNotifyQueue
	// CMDAddOrUpdateConversations CMDAddOrUpdateConversations
	CMDAddOrUpdateConversations
	// CMDAddDenylist CMDAddDenylist
	CMDAddDenylist
	// CMDRemoveDenylist CMDRemoveDenylist
	CMDRemoveDenylist
	// CMDRemoveAllDenylist CMDRemoveAllDenylist
	CMDRemoveAllDenylist
	// CMDAddAllowlist CMDAddAllowlist
	CMDAddAllowlist
	// CMDRemoveAllowlist CMDRemoveAllowlist
	CMDRemoveAllowlist
	// CMDRemoveAllAllowlist CMDRemoveAllAllowlist
	CMDRemoveAllAllowlist
	// CMDAddNodeInFlightData CMDAddNodeInFlightData
	CMDAddNodeInFlightData
	// CMDClearNodeInFlightData CMDClearNodeInFlightData
	CMDClearNodeInFlightData
	// CMDUpdateMessageOfUserCursorIfNeed CMDUpdateMessageOfUserCursorIfNeed
	CMDUpdateMessageOfUserCursorIfNeed
)

// Int32 Int32
func (c CMDType) Int32() int32 {
	return int32(c)
}
func (c CMDType) String() string {
	switch c {
	case CMDAppendMessage:
		return "CMDAppendMessage"
	case CMDAddOrUpdateChannel:
		return "CMDAddOrUpdateChannel"
	case CMDUpdateUserToken:
		return "CMDUpdateUserToken"
	case CMDRemoveAllSubscriber:
		return "CMDRemoveAllSubscriber"
	case CMDRemoveSubscribers:
		return "CMDRemoveSubscribers"
	case CMDAddSubscribers:
		return "CMDAddSubscribers"
	case CMDDeleteChannel:
		return "CMDDeleteChannel"
	case CMDDeleteChannelAndClearMessages:
		return "CMDDeleteChannelAndClearMessages"
	case CMDAppendMessageOfUser:
		return "CMDAppendMessageOfUser"
	case CMDAppendMessageOfNotifyQueue:
		return "CMDAppendMessageOfNotifyQueue"
	case CMDRemoveMessagesOfNotifyQueue:
		return "CMDRemoveMessagesOfNotifyQueue"
	case CMDAddOrUpdateConversations:
		return "CMDAddOrUpdateConversations"
	case CMDAddDenylist:
		return "CMDAddDenylist"
	case CMDRemoveDenylist:
		return "CMDRemoveDenylist"
	case CMDRemoveAllDenylist:
		return "CMDRemoveAllDenylist"
	case CMDAddAllowlist:
		return "CMDAddAllowlist"
	case CMDRemoveAllAllowlist:
		return "CMDRemoveAllAllowlist"
	case CMDAddNodeInFlightData:
		return "CMDAddNodeInFlightData"
	case CMDClearNodeInFlightData:
		return "CMDClearNodeInFlightData"
	case CMDUpdateMessageOfUserCursorIfNeed:
		return "CMDUpdateMessageOfUserCursorIfNeed"
	}
	return fmt.Sprintf("%d", c)
}

// CMD CMD
type CMD struct {
	Type    CMDType
	Version uint8
	Param   []byte
}

// NewCMD NewCMD
func NewCMD(typ CMDType) *CMD {
	return &CMD{
		Type: typ,
	}
}

func (c *CMD) String() string {
	return fmt.Sprintf("Type:%s Version: %d Param: %s", c.Type.String(), c.Version, fmt.Sprintf("Size(%d)", len(c.Param)))
}

// Encode Encode
func (c *CMD) Encode() []byte {
	enc := lmproto.NewEncoder()
	enc.WriteInt32(c.Type.Int32())
	enc.WriteUint8(c.Version)
	enc.WriteBytes(c.Param)
	return enc.Bytes()

}

// UnmarshalCMD UnmarshalCMD
func UnmarshalCMD(data []byte, cmd *CMD) error {
	dec := lmproto.NewDecoder(data)
	var err error
	var cmdType int32
	if cmdType, err = dec.Int32(); err != nil {
		return err
	}
	cmd.Type = CMDType(cmdType)

	if cmd.Version, err = dec.Uint8(); err != nil {
		return err
	}

	if cmd.Param, err = dec.BinaryAll(); err != nil {
		return err
	}
	return nil
}

// EncodeUserToken EncodeUserToken
func (c *CMD) EncodeUserToken(uid string, deviceFlag lmproto.DeviceFlag, deviceLevel lmproto.DeviceLevel, token string) *CMD {
	c.Type = CMDUpdateUserToken
	encoder := lmproto.NewEncoder()
	encoder.WriteString(uid)
	encoder.WriteUint8(deviceFlag.ToUint8())
	encoder.WriteUint8(uint8(deviceLevel))
	encoder.WriteString(token)
	c.Param = encoder.Bytes()
	return c
}

// DecodeUserToken DecodeUserToken
func (c *CMD) DecodeUserToken() (uid string, deviceFlag lmproto.DeviceFlag, deviceLevel lmproto.DeviceLevel, token string, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if uid, err = decoder.String(); err != nil {
		return
	}
	var deviceFlagI uint8
	if deviceFlagI, err = decoder.Uint8(); err != nil {
		return
	}

	deviceFlag = lmproto.DeviceFlag(deviceFlagI)

	var deviceLevelI uint8
	if deviceLevelI, err = decoder.Uint8(); err != nil {
		return
	}

	deviceLevel = lmproto.DeviceLevel(deviceLevelI)

	if token, err = decoder.String(); err != nil {
		return
	}
	return
}

// EncodeAddOrUpdateChannel EncodeAddOrUpdateChannel
func (c *CMD) EncodeAddOrUpdateChannel(channelInfo *ChannelInfo) *CMD {
	c.Type = CMDAddOrUpdateChannel
	encoder := lmproto.NewEncoder()
	encoder.WriteString(channelInfo.ChannelID)
	encoder.WriteUint8(channelInfo.ChannelType)
	encoder.WriteString(util.ToJSON(channelInfo.ToMap()))
	c.Param = encoder.Bytes()
	return c
}

// DecodeAddOrUpdateChannel DecodeAddOrUpdateChannel
func (c *CMD) DecodeAddOrUpdateChannel() (*ChannelInfo, error) {
	decoder := lmproto.NewDecoder(c.Param)
	channelInfo := &ChannelInfo{}
	var err error
	if channelInfo.ChannelID, err = decoder.String(); err != nil {
		return nil, err
	}
	if channelInfo.ChannelType, err = decoder.Uint8(); err != nil {
		return nil, err
	}
	jsonStr, err := decoder.String()
	if err != nil {
		return nil, err
	}
	if len(jsonStr) > 0 {
		mp, err := util.JSONToMap(jsonStr)
		if err != nil {
			return nil, err
		}
		channelInfo.from(mp)
	}
	return channelInfo, nil
}

// EncodeCMDAddSubscribers EncodeCMDAddSubscribers
func (c *CMD) EncodeCMDAddSubscribers(channelID string, channelType uint8, uids []string) *CMD {
	c.Type = CMDAddSubscribers
	encoder := lmproto.NewEncoder()
	encoder.WriteString(channelID)
	encoder.WriteUint8(channelType)
	if len(uids) > 0 {
		encoder.WriteString(util.ToJSON(uids))
	}
	c.Param = encoder.Bytes()
	return c
}

// DecodeCMDAddSubscribers DecodeCMDAddSubscribers
func (c *CMD) DecodeCMDAddSubscribers() (channelID string, channelType uint8, uids []string, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if channelID, err = decoder.String(); err != nil {
		return
	}
	if channelType, err = decoder.Uint8(); err != nil {
		return
	}
	var uidsBytes []byte
	uidsBytes, err = decoder.Binary()
	if err != nil {
		return
	}
	if len(uidsBytes) > 0 {
		err = util.ReadJSONByByte(uidsBytes, &uids)
		if err != nil {
			return
		}
	}
	return
}

// EncodeRemoveAllSubscriber EncodeRemoveAllSubscriber
func (c *CMD) EncodeRemoveAllSubscriber(channelID string, channelType uint8) *CMD {
	c.Type = CMDRemoveAllSubscriber
	encoder := lmproto.NewEncoder()
	encoder.WriteString(channelID)
	encoder.WriteUint8(channelType)
	c.Param = encoder.Bytes()
	return c
}

// DecodeRemoveAllSubscriber DecodeRemoveAllSubscriber
func (c *CMD) DecodeRemoveAllSubscriber() (channelID string, channelType uint8, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if channelID, err = decoder.String(); err != nil {
		return
	}
	if channelType, err = decoder.Uint8(); err != nil {
		return
	}
	return
}

// EncodeCMDRemoveSubscribers EncodeCMDRemoveSubscribers
func (c *CMD) EncodeCMDRemoveSubscribers(channelID string, channelType uint8, uids []string) *CMD {
	c.Type = CMDRemoveSubscribers
	encoder := lmproto.NewEncoder()
	encoder.WriteString(channelID)
	encoder.WriteUint8(channelType)
	if len(uids) > 0 {
		encoder.WriteString(util.ToJSON(uids))
	}
	c.Param = encoder.Bytes()
	return c
}

// DecodeRemoveSubscribers DecodeRemoveSubscribers
func (c *CMD) DecodeRemoveSubscribers() (channelID string, channelType uint8, uids []string, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if channelID, err = decoder.String(); err != nil {
		return
	}
	if channelType, err = decoder.Uint8(); err != nil {
		return
	}
	var uidsBytes []byte
	uidsBytes, err = decoder.Binary()
	if err != nil {
		return
	}
	if len(uidsBytes) > 0 {
		err = util.ReadJSONByByte(uidsBytes, &uids)
		if err != nil {
			return
		}
	}
	return
}

// EncodeCMDDeleteChannel EncodeCMDDeleteChannel
func (c *CMD) EncodeCMDDeleteChannel(channelID string, channelType uint8) *CMD {
	c.Type = CMDDeleteChannel
	encoder := lmproto.NewEncoder()
	encoder.WriteString(channelID)
	encoder.WriteUint8(channelType)
	c.Param = encoder.Bytes()
	return c
}

// DecodeCMDDeleteChannel DecodeCMDDeleteChannel
func (c *CMD) DecodeCMDDeleteChannel() (channelID string, channelType uint8, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if channelID, err = decoder.String(); err != nil {
		return
	}
	if channelType, err = decoder.Uint8(); err != nil {
		return
	}
	return
}

// EncodeCMDDeleteChannelAndClearMessages EncodeCMDDeleteChannelAndClearMessages
func (c *CMD) EncodeCMDDeleteChannelAndClearMessages(channelID string, channelType uint8) *CMD {
	c.Type = CMDDeleteChannelAndClearMessages
	encoder := lmproto.NewEncoder()
	encoder.WriteString(channelID)
	encoder.WriteUint8(channelType)
	c.Param = encoder.Bytes()
	return c
}

// DecodeCMDDeleteChannelAndClearMessages DecodeCMDDeleteChannelAndClearMessages
func (c *CMD) DecodeCMDDeleteChannelAndClearMessages() (channelID string, channelType uint8, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if channelID, err = decoder.String(); err != nil {
		return
	}
	if channelType, err = decoder.Uint8(); err != nil {
		return
	}
	return
}

// EncodeAppendMessage EncodeAppendMessage
func (c *CMD) EncodeAppendMessage(m *db.Message) *CMD {
	c.Type = CMDAppendMessage
	c.Param = db.MarshalMessage(m)
	return c
}

// DecodeAppendMessage DecodeAppendMessage
func (c *CMD) DecodeAppendMessage() (*db.Message, error) {
	m := &db.Message{}
	err := db.UnmarshalMessage(c.Param, m)
	return m, err
}

// EncodeCMDAppendMessageOfUser EncodeCMDAppendMessageOfUser
func (c *CMD) EncodeCMDAppendMessageOfUser(m *db.Message) *CMD {
	c.Type = CMDAppendMessageOfUser
	encoder := lmproto.NewEncoder()
	encoder.WriteBytes(db.MarshalMessage(m))
	c.Param = encoder.Bytes()
	return c
}

// DecodeCMDAppendMessageOfUser DecodeCMDAppendMessageOfUser
func (c *CMD) DecodeCMDAppendMessageOfUser() (m *db.Message, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	var messageData []byte
	if messageData, err = decoder.BinaryAll(); err != nil {
		return
	}
	m = &db.Message{}
	if err = db.UnmarshalMessage(messageData, m); err != nil {
		return
	}
	return
}

// EncodeCMDUpdateMessageOfUserCursorIfNeed EncodeCMDUpdateMessageOfUserCursorIfNeed
func (c *CMD) EncodeCMDUpdateMessageOfUserCursorIfNeed(uid string, messageSeq uint32) *CMD {
	c.Type = CMDUpdateMessageOfUserCursorIfNeed
	encoder := lmproto.NewEncoder()
	encoder.WriteString(uid)
	encoder.WriteUint32(messageSeq)
	c.Param = encoder.Bytes()
	return c
}

// DecodeCMDUpdateMessageOfUserCursorIfNeed DecodeCMDUpdateMessageOfUserCursorIfNeed
func (c *CMD) DecodeCMDUpdateMessageOfUserCursorIfNeed() (uid string, messageSeq uint32, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if uid, err = decoder.String(); err != nil {
		return
	}
	if messageSeq, err = decoder.Uint32(); err != nil {
		return
	}
	return
}

// EncodeCMDAppendMessageOfNotifyQueue EncodeCMDAppendMessageOfNotifyQueue
func (c *CMD) EncodeCMDAppendMessageOfNotifyQueue(m *db.Message) *CMD {
	c.Type = CMDAppendMessageOfNotifyQueue
	c.Param = db.MarshalMessage(m)
	return c
}

// DecodeCMDAppendMessageOfNotifyQueue DecodeCMDAppendMessageOfNotifyQueue
func (c *CMD) DecodeCMDAppendMessageOfNotifyQueue() (*db.Message, error) {
	m := &db.Message{}
	err := db.UnmarshalMessage(c.Param, m)
	return m, err
}

// EncodeCMDRemoveMessagesOfNotifyQueue EncodeCMDRemoveMessagesOfNotifyQueue
func (c *CMD) EncodeCMDRemoveMessagesOfNotifyQueue(messageIDs []int64) *CMD {
	c.Type = CMDRemoveMessagesOfNotifyQueue
	c.Param = []byte(util.ToJSON(messageIDs))
	return c
}

// DecodeCMDRemoveMessagesOfNotifyQueue DecodeCMDRemoveMessagesOfNotifyQueue
func (c *CMD) DecodeCMDRemoveMessagesOfNotifyQueue() ([]int64, error) {
	var messageIDs []int64
	if err := util.ReadJSONByByte(c.Param, &messageIDs); err != nil {
		return nil, err
	}
	return messageIDs, nil
}

// EncodeCMDAddOrUpdateConversations EncodeCMDAddOrUpdateConversations
func (c *CMD) EncodeCMDAddOrUpdateConversations(uid string, conversations []*db.Conversation) *CMD {
	c.Type = CMDAddOrUpdateConversations
	encoder := lmproto.NewEncoder()
	encoder.WriteString(uid)
	encoder.WriteString(util.ToJSON(conversations))

	c.Param = encoder.Bytes()
	return c
}

// DecodeCMDAddOrUpdateConversations DecodeCMDAddOrUpdateConversations
func (c *CMD) DecodeCMDAddOrUpdateConversations() (uid string, conversations []*db.Conversation, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if uid, err = decoder.String(); err != nil {
		return
	}
	var data []byte
	if data, err = decoder.Binary(); err != nil {
		return
	}
	if len(data) > 0 {
		if err = util.ReadJSONByByte(data, &conversations); err != nil {
			return
		}
	}
	return
}

// EncodeCMDAddDenylist EncodeCMDAddDenylist
func (c *CMD) EncodeCMDAddDenylist(channelID string, channelType uint8, uids []string) *CMD {
	c.Type = CMDAddDenylist
	encoder := lmproto.NewEncoder()
	encoder.WriteString(channelID)
	encoder.WriteUint8(channelType)
	if len(uids) > 0 {
		encoder.WriteString(util.ToJSON(uids))
	}
	c.Param = encoder.Bytes()
	return c
}

// DecodeCMDAddDenylist DecodeCMDAddDenylist
func (c *CMD) DecodeCMDAddDenylist() (channelID string, channelType uint8, uids []string, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if channelID, err = decoder.String(); err != nil {
		return
	}
	if channelType, err = decoder.Uint8(); err != nil {
		return
	}
	var uidsBytes []byte
	uidsBytes, err = decoder.Binary()
	if err != nil {
		return
	}
	if len(uidsBytes) > 0 {
		err = util.ReadJSONByByte(uidsBytes, &uids)
		if err != nil {
			return
		}
	}
	return
}

// EncodeCMDRemoveDenylist EncodeCMDRemoveDenylist
func (c *CMD) EncodeCMDRemoveDenylist(channelID string, channelType uint8, uids []string) *CMD {
	c.Type = CMDRemoveDenylist
	encoder := lmproto.NewEncoder()
	encoder.WriteString(channelID)
	encoder.WriteUint8(channelType)
	if len(uids) > 0 {
		encoder.WriteString(util.ToJSON(uids))
	}
	c.Param = encoder.Bytes()
	return c
}

// DecodeCMDRemoveDenylist DecodeCMDRemoveDenylist
func (c *CMD) DecodeCMDRemoveDenylist() (channelID string, channelType uint8, uids []string, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if channelID, err = decoder.String(); err != nil {
		return
	}
	if channelType, err = decoder.Uint8(); err != nil {
		return
	}
	var uidsBytes []byte
	uidsBytes, err = decoder.Binary()
	if err != nil {
		return
	}
	if len(uidsBytes) > 0 {
		err = util.ReadJSONByByte(uidsBytes, &uids)
		if err != nil {
			return
		}
	}
	return
}

// EncodeCMDRemoveAllDenylist EncodeCMDRemoveAllDenylist
func (c *CMD) EncodeCMDRemoveAllDenylist(channelID string, channelType uint8) *CMD {
	c.Type = CMDRemoveAllDenylist
	encoder := lmproto.NewEncoder()
	encoder.WriteString(channelID)
	encoder.WriteUint8(channelType)
	c.Param = encoder.Bytes()
	return c
}

// DecodeCMDRemoveAllDenylist DecodeCMDRemoveAllDenylist
func (c *CMD) DecodeCMDRemoveAllDenylist() (channelID string, channelType uint8, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if channelID, err = decoder.String(); err != nil {
		return
	}
	if channelType, err = decoder.Uint8(); err != nil {
		return
	}
	return
}

// EncodeCMDAddAllowlist EncodeCMDAddAllowlist
func (c *CMD) EncodeCMDAddAllowlist(channelID string, channelType uint8, uids []string) *CMD {
	c.Type = CMDAddAllowlist
	encoder := lmproto.NewEncoder()
	encoder.WriteString(channelID)
	encoder.WriteUint8(channelType)
	if len(uids) > 0 {
		encoder.WriteString(util.ToJSON(uids))
	}
	c.Param = encoder.Bytes()
	return c
}

// DecodeCMDAddAllowlist DecodeCMDAddAllowlist
func (c *CMD) DecodeCMDAddAllowlist() (channelID string, channelType uint8, uids []string, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if channelID, err = decoder.String(); err != nil {
		return
	}
	if channelType, err = decoder.Uint8(); err != nil {
		return
	}
	var uidsBytes []byte
	uidsBytes, err = decoder.Binary()
	if err != nil {
		return
	}
	if len(uidsBytes) > 0 {
		err = util.ReadJSONByByte(uidsBytes, &uids)
		if err != nil {
			return
		}
	}
	return
}

// EncodeCMDRemoveAllowlist EncodeCMDRemoveAllowlist
func (c *CMD) EncodeCMDRemoveAllowlist(channelID string, channelType uint8, uids []string) *CMD {
	c.Type = CMDRemoveAllowlist
	encoder := lmproto.NewEncoder()
	encoder.WriteString(channelID)
	encoder.WriteUint8(channelType)
	if len(uids) > 0 {
		encoder.WriteString(util.ToJSON(uids))
	}
	c.Param = encoder.Bytes()
	return c
}

// DecodeCMDRemoveAllowlist DecodeCMDRemoveAllowlist
func (c *CMD) DecodeCMDRemoveAllowlist() (channelID string, channelType uint8, uids []string, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if channelID, err = decoder.String(); err != nil {
		return
	}
	if channelType, err = decoder.Uint8(); err != nil {
		return
	}
	var uidsBytes []byte
	uidsBytes, err = decoder.Binary()
	if err != nil {
		return
	}
	if len(uidsBytes) > 0 {
		err = util.ReadJSONByByte(uidsBytes, &uids)
		if err != nil {
			return
		}
	}
	return
}

// EncodeCMDRemoveAllAllowlist EncodeCMDRemoveAllAllowlist
func (c *CMD) EncodeCMDRemoveAllAllowlist(channelID string, channelType uint8) *CMD {
	c.Type = CMDRemoveAllAllowlist
	encoder := lmproto.NewEncoder()
	encoder.WriteString(channelID)
	encoder.WriteUint8(channelType)
	c.Param = encoder.Bytes()
	return c
}

// DecodeCMDRemoveAllAllowlist DecodeCMDRemoveAllAllowlist
func (c *CMD) DecodeCMDRemoveAllAllowlist() (channelID string, channelType uint8, err error) {
	decoder := lmproto.NewDecoder(c.Param)
	if channelID, err = decoder.String(); err != nil {
		return
	}
	if channelType, err = decoder.Uint8(); err != nil {
		return
	}
	return
}

// EncodeCMDAddNodeInFlightData EncodeCMDAddNodeInFlightData
func (c *CMD) EncodeCMDAddNodeInFlightData(data []*db.NodeInFlightDataModel) *CMD {
	c.Type = CMDAddNodeInFlightData
	c.Param = []byte(util.ToJSON(data))
	return c
}

// DecodeCMDAddNodeInFlightData DecodeCMDAddNodeInFlightData
func (c *CMD) DecodeCMDAddNodeInFlightData() ([]*db.NodeInFlightDataModel, error) {
	var inflightDatas []*db.NodeInFlightDataModel
	if err := util.ReadJSONByByte(c.Param, &inflightDatas); err != nil {
		return nil, err
	}
	return inflightDatas, nil
}

// EncodeCMDClearNodeInFlightData EncodeCMDClearNodeInFlightData
func (c *CMD) EncodeCMDClearNodeInFlightData() *CMD {
	c.Type = CMDClearNodeInFlightData
	return c
}
