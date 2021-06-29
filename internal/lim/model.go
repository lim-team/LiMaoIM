package lim

import (
	"fmt"

	"github.com/lim-team/LiMaoIM/pkg/lmproto"
)

// CMDType CMDType
type CMDType int

// Int32 Int32
func (c CMDType) Int32() int32 {
	return int32(c)
}
func (c CMDType) String() string {
	switch c {
	case CMDAppendMessage:
		return "CMDAppendMessage"
	case CMDAddChannel:
		return "CMDAddChannel"
	}
	return "CMDUnknown"
}

const (
	// CMDUnknown unknown
	CMDUnknown = 0
	// CMDAppendMessage append message
	CMDAppendMessage = 1
	// CMDAddChannel add channel
	CMDAddChannel = 2
)

// CMD CMD
type CMD struct {
	ID    uint64
	Type  CMDType
	Param []byte
}

func (c *CMD) String() string {
	return fmt.Sprintf("ID: %d Type:%s Param: %s", c.ID, c.Type.String(), string(c.Param))
}

// MarshalCMD MarshalCMD
func MarshalCMD(cmd *CMD) ([]byte, error) {
	enc := lmproto.NewEncoder()
	enc.WriteUint64(cmd.ID)
	enc.WriteInt32(cmd.Type.Int32())
	enc.WriteBytes(cmd.Param)

	return enc.Bytes(), nil
}

// UnmarshalCMD UnmarshalCMD
func UnmarshalCMD(data []byte, cmd *CMD) error {
	dec := lmproto.NewDecoder(data)
	var err error
	if cmd.ID, err = dec.Uint64(); err != nil {
		return err
	}
	var cmdType int32
	if cmdType, err = dec.Int32(); err != nil {
		return err
	}
	cmd.Type = CMDType(cmdType)
	if cmd.Param, err = dec.BinaryAll(); err != nil {
		return err
	}
	return nil
}
