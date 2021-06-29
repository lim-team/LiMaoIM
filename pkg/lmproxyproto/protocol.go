package lmproxyproto

import (
	"fmt"
	"io"
)

// CMDType cmd类型
type CMDType uint8

const (
	// CMDTypeNone 无
	CMDTypeNone CMDType = iota
	// CMDTypeConnect 连接
	CMDTypeConnect
	// CMDTypeConnectResp 连接回执
	CMDTypeConnectResp
	// CMDTypeClusterConfigReq 配置请求
	CMDTypeClusterConfigReq
	// CMDTypeClusterConfigResp 配置返回
	CMDTypeClusterConfigResp
	// CMDTypeClusterConfigUpdate 配置更新
	CMDTypeClusterConfigUpdate
	// CMDTypeUploadClusterConfig 代理服务命令节点上传分布式配置
	CMDTypeUploadClusterConfig
	// CMDTypeClusterConfigUpload 节点上传配置
	CMDTypeClusterConfigUpload
	// CMDTypeRegisterNode 注册节点
	CMDTypeRegisterNode
	// CMDTypeRegisterNodeResp 注册节点返回
	CMDTypeRegisterNodeResp
	// CMDTypePing 心跳
	CMDTypePing
	// CMDTypePong 心跳响应
	CMDTypePong
	// CMDTypeStatusResp CMDTypeStatusResp
	CMDTypeStatusResp
	// CMDTypeExporting CMDTypeExporting
	CMDTypeExporting
	// CMDTypeExported CMDTypeExported
	CMDTypeExported
	// CMDTypeExportedAll CMDTypeExportedAll
	CMDTypeExportedAll
	// CMDTypeImportFrom CMDTypeImportFrom
	CMDTypeImportFrom
	// CMDTypeImported CMDTypeImported
	CMDTypeImported
	// CMDTypeClusterConfigListReq 请求分布式配置列表 （n-p）
	CMDTypeClusterConfigListReq
	// CMDTypeClusterConfigListResp 分布式配置列表返回 （p-n）
	CMDTypeClusterConfigListResp
	// CMDTypeClusterConfigListChange 分布配置列表有变动
	CMDTypeClusterConfigListChange
)

// Uint8 Uint8
func (c CMDType) Uint8() uint8 {
	return uint8(c)
}

func (c CMDType) String() string {
	switch c {
	case CMDTypeConnect:
		return "CMDTypeConnect"
	case CMDTypeConnectResp:
		return "CMDTypeConnectResp"
	case CMDTypeClusterConfigReq:
		return "CMDTypeClusterConfigReq"
	case CMDTypeClusterConfigResp:
		return "CMDTypeClusterConfigResp"
	case CMDTypeClusterConfigUpdate:
		return "CMDTypeClusterConfigUpdate"
	case CMDTypeUploadClusterConfig:
		return "CMDTypeUploadClusterConfig"
	case CMDTypeClusterConfigUpload:
		return "CMDTypeClusterConfigUpload"
	case CMDTypeRegisterNode:
		return "CMDTypeRegisterNode"
	case CMDTypeRegisterNodeResp:
		return "CMDTypeRegisterNodeResp"
	case CMDTypePing:
		return "CMDTypePing"
	case CMDTypePong:
		return "CMDTypePong"
	case CMDTypeStatusResp:
		return "CMDTypeStatusResp"
	case CMDTypeExporting:
		return "CMDTypeExporting"
	case CMDTypeExported:
		return "CMDTypeExported"
	case CMDTypeExportedAll:
		return "CMDTypeExportedAll"
	case CMDTypeImportFrom:
		return "CMDTypeImportFrom"
	case CMDTypeImported:
		return "CMDTypeImported"
	case CMDTypeClusterConfigListReq:
		return "CMDTypeClusterConfigListReq"
	case CMDTypeClusterConfigListResp:
		return "CMDTypeClusterConfigListResp"
	case CMDTypeClusterConfigListChange:
		return "CMDTypeClusterConfigListChange"
	}
	return ""
}

// CMD CMD
type CMD struct {
	Cmd   CMDType // cmd类型
	ID    uint64
	Param []byte // cmd参数
}

func (c *CMD) String() string {
	return fmt.Sprintf("ID: %d CMD: %s", c.ID, c.Cmd)
}

// Protocol 协议
type Protocol struct {
}

// NewProtocol 创建一个协议
func NewProtocol() *Protocol {
	return &Protocol{}
}

// Encode 编码
func (p *Protocol) Encode(c *CMD) ([]byte, error) {
	enc := NewEncoder()
	enc.WriteUint8(c.Cmd.Uint8())
	enc.WriteUint64(c.ID)
	enc.WriteInt32(int32(len(c.Param)))
	enc.WriteBytes(c.Param)
	return enc.Bytes(), nil
}

// Decode 解码
func (p *Protocol) Decode(data []byte) (*CMD, error) {
	cmd := &CMD{}
	cmd.Cmd = CMDType(uint8(data[0]))
	cmd.ID = p.Uint64(data[1:9])
	paramLen := p.Int32(data[9 : 9+4])
	cmd.Param = data[9+4 : paramLen+9+4]
	return cmd, nil
}

// DecodeWithReader 解码
func (p *Protocol) DecodeWithReader(reader io.Reader) (*CMD, error) {
	cmdBytes := make([]byte, 5+8)
	_, err := reader.Read(cmdBytes)
	if err != nil {
		return nil, err
	}
	cmd := uint8(cmdBytes[0])
	id := p.Uint64(cmdBytes[1:9])
	len := p.Int32(cmdBytes[9:])
	data := make([]byte, len)
	_, err = reader.Read(data)
	if err != nil {
		return nil, err
	}
	return &CMD{
		ID:    id,
		Cmd:   CMDType(cmd),
		Param: data,
	}, nil
}

// Int32 Int32
func (p *Protocol) Int32(b []byte) int32 {
	return (int32(b[0]) << 24) | (int32(b[1]) << 16) | (int32(b[2]) << 8) | int32(b[3])
}

// Uint64 Uint64
func (p *Protocol) Uint64(b []byte) uint64 {
	return (uint64(b[0]) << 56) | (uint64(b[1]) << 48) | (uint64(b[2]) << 40) | uint64(b[3])<<32 | uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
}
