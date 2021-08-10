package rpc

import (
	context "context"
	"log"
	"net"

	"github.com/golang/protobuf/proto"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"go.uber.org/zap"
	grpc "google.golang.org/grpc"
)

const (
	// CMDTypeForwardSendPacket 转发发送包
	CMDTypeForwardSendPacket = 1001
)

// CMDEvent cmd事件
type CMDEvent interface {
	// 收到发送包
	OnSendPacket(fromUID string, deviceFlag lmproto.DeviceFlag, sendPacket *lmproto.SendPacket) (messageID int64, messageSeq uint32, reasonCode lmproto.ReasonCode, err error)
	// 收到接受包
	OnRecvPacket(deviceFlag lmproto.DeviceFlag, recvPacket *lmproto.RecvPacket, users []string) error
	// 获取频道订阅者
	OnGetSubscribers(channelID string, channelType uint8) ([]string, error)
	// 获取频道的消息序号
	OnGetChannelMessageSeq(channelID string, channelType uint8) (uint32, error)
}

// Server rpc服务
type Server struct {
	event CMDEvent
	proto lmproto.Protocol
	limlog.Log
	addr string
}

// NewServer NewServer
func NewServer(proto lmproto.Protocol, event CMDEvent, addr string) *Server {
	return &Server{
		event: event,
		proto: proto,
		Log:   limlog.NewLIMLog("NodeServer"),
		addr:  addr,
	}
}

// Start 开启rpc服务
func (s *Server) Start() {
	grpcServer := grpc.NewServer()
	RegisterNodeServiceServer(grpcServer, &nodeServiceImp{s: s})

	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Fatal(err)
	}
	go grpcServer.Serve(lis)
}

type nodeServiceImp struct {
	s *Server
	UnimplementedNodeServiceServer
}

func (n *nodeServiceImp) SendCMD(ctx context.Context, req *CMDReq) (*CMDResp, error) {
	n.s.Debug("收到命令", zap.String("cmd", req.Cmd.String()))
	if req.Cmd == CMDType_ForwardSendPacket { //收到转发发送包
		forwardSendPacketReq := &ForwardSendPacketReq{}
		err := proto.Unmarshal(req.Data, forwardSendPacketReq)
		if err != nil {
			return nil, err
		}
		sendPacket, _, err := n.s.proto.DecodePacket(forwardSendPacketReq.SendPacket, lmproto.LatestVersion)
		if err != nil {
			return nil, err
		}
		messageID, messageSeq, reasonCode, err := n.s.event.OnSendPacket(forwardSendPacketReq.FromUID, lmproto.DeviceFlag(forwardSendPacketReq.FromDeviceFlag), sendPacket.(*lmproto.SendPacket))
		if err != nil {
			return nil, err
		}
		resp := &ForwardSendPacketResp{
			MessageID:  messageID,
			MessageSeq: int32(messageSeq),
			ReasonCode: int32(reasonCode),
		}
		respData, err := proto.Marshal(resp)
		if err != nil {
			return nil, err
		}
		return &CMDResp{
			Status: Status_Success,
			Data:   respData,
		}, nil
	} else if req.Cmd == CMDType_ForwardRecvPacket { // 收到转发接受包
		forwardRecvPacketReq := &ForwardRecvPacketReq{}
		err := proto.Unmarshal(req.Data, forwardRecvPacketReq)
		if err != nil {
			n.s.Error("解码转发接受包数据失败！", zap.Error(err))
			return nil, err
		}
		recvPacket, _, err := n.s.proto.DecodePacket(forwardRecvPacketReq.Message, lmproto.LatestVersion)
		if err != nil {
			n.s.Error("解码接受包数据失败！", zap.Error(err))
			return nil, err
		}
		err = n.s.event.OnRecvPacket(lmproto.DeviceFlag(forwardRecvPacketReq.FromDeviceFlag), recvPacket.(*lmproto.RecvPacket), forwardRecvPacketReq.Users)
		if err != nil {
			return nil, err
		}
		return &CMDResp{
			Status: Status_Success,
		}, nil
	} else if req.Cmd == CMDType_GetSubscribers { // 获取订阅者
		getSubscribersReq := &GetSubscribersReq{}
		err := proto.Unmarshal(req.Data, getSubscribersReq)
		if err != nil {
			n.s.Error("解码获取订阅者请求数据失败！", zap.Error(err))
			return nil, err
		}
		subscribers, err := n.s.event.OnGetSubscribers(getSubscribersReq.ChannelID, uint8(getSubscribersReq.ChannelType))
		if err != nil {
			return nil, err
		}
		respData, err := proto.Marshal(&GetSubscribersResp{
			Subscribers: subscribers,
		})
		if err != nil {
			return nil, err
		}
		return &CMDResp{
			Status: Status_Success,
			Data:   respData,
		}, nil
	} else if req.Cmd == CMDType_GenChannelMessageSeq {
		getChannelMessageSeqReq := &GetChannelMessageSeqReq{}
		err := proto.Unmarshal(req.Data, getChannelMessageSeqReq)
		if err != nil {
			n.s.Error("解码获取频道消息序号请求数据失败！", zap.Error(err))
			return nil, err
		}
		messageSeq, err := n.s.event.OnGetChannelMessageSeq(getChannelMessageSeqReq.ChannelID, uint8(getChannelMessageSeqReq.ChannelType))
		if err != nil {
			return nil, err
		}
		respData, err := proto.Marshal(&GetChannelMessageSeqResp{
			MessageSeq: int32(messageSeq),
		})
		if err != nil {
			return nil, err
		}
		return &CMDResp{
			Status: Status_Success,
			Data:   respData,
		}, nil
	} else {
		n.s.Error("不支持的RPC CMD", zap.String("cmd", req.Cmd.String()))
		return &CMDResp{
			Status: Status_Error,
		}, nil
	}
}
