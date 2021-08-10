package lim

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/lim-team/LiMaoIM/internal/lim/rpc"
	"github.com/lim-team/LiMaoIM/pkg/limlog"
	"github.com/lim-team/LiMaoIM/pkg/lmproxyproto"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// NodeRemoteCall NodeRemoteCall
type NodeRemoteCall struct {
	limlog.Log
	l           *LiMao
	callTimeout time.Duration
}

// NewNodeRemoteCall NewNodeRemoteCall
func NewNodeRemoteCall(l *LiMao) *NodeRemoteCall {

	return &NodeRemoteCall{
		l:           l,
		Log:         limlog.NewLIMLog("NodeRemoteCall"),
		callTimeout: time.Second * 10,
	}
}

func (n *NodeRemoteCall) getNodeByID(nodeID int32) *lmproxyproto.Node {
	return n.l.clusterManager.GetNodeWithID(nodeID)
}

func (n *NodeRemoteCall) getNodeConn(nodeID int32) (*grpc.ClientConn, error) {
	node := n.getNodeByID(nodeID)
	if node == nil {
		n.Error("节点不存在！", zap.Int32("nodeID", nodeID))
		return nil, errors.New("节点不存在！")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second))
	defer cancel()
	conn, err := grpc.DialContext(ctx, node.NodeRPCAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// SendCMD 发送cmd给指定节点
func (n *NodeRemoteCall) SendCMD(ctx context.Context, nodeID int32, cmd *rpc.CMDReq) (*rpc.CMDResp, error) {
	conn, err := n.getNodeConn(nodeID)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := rpc.NewNodeServiceClient(conn)
	return client.SendCMD(ctx, cmd)
}

// GetChannelMessageSeq 获取频道消息序号
func (n *NodeRemoteCall) GetChannelMessageSeq(channelID string, channelType uint8, nodeID int32) (uint32, error) {
	reqData, _ := proto.Marshal(&rpc.GetChannelMessageSeqReq{
		ChannelID:   channelID,
		ChannelType: int32(channelType),
	})
	timeoutCtx, cancel := context.WithTimeout(context.Background(), n.callTimeout)
	resp, err := n.SendCMD(timeoutCtx, nodeID, &rpc.CMDReq{
		Cmd:  rpc.CMDType_GenChannelMessageSeq,
		Data: reqData,
	})
	cancel()
	if err != nil {
		n.Error("获取频道序号请求失败！", zap.Error(err))
		return 0, err
	}

	getChannelMessageSeqResp := &rpc.GetChannelMessageSeqResp{}
	err = proto.Unmarshal(resp.Data, getChannelMessageSeqResp)
	if err != nil {
		return 0, err
	}
	return uint32(getChannelMessageSeqResp.MessageSeq), nil
}

// ForwardSendPacket 转发发送包
func (n *NodeRemoteCall) ForwardSendPacket(req *rpc.ForwardSendPacketReq, nodeID int32) (*rpc.ForwardSendPacketResp, error) {

	reqData, _ := proto.Marshal(req)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), n.callTimeout)
	resp, err := n.SendCMD(timeoutCtx, nodeID, &rpc.CMDReq{
		Cmd:  rpc.CMDType_ForwardSendPacket,
		Data: reqData,
	})
	cancel()
	if err != nil {
		n.Error("转发发送包失败！", zap.Error(err))
		return nil, err
	}
	if resp.Status != rpc.Status_Success {
		return nil, errors.New("转发发送包返回结果失败！")
	}
	sendResp := &rpc.ForwardSendPacketResp{}
	err = proto.Unmarshal(resp.Data, sendResp)
	if err != nil {
		return nil, err
	}
	return sendResp, nil
}

// ForwardRecvPacket 转发消息
func (n *NodeRemoteCall) ForwardRecvPacket(req *rpc.ForwardRecvPacketReq, nodeID int32) error {
	data, _ := proto.Marshal(req)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), n.callTimeout)
	resp, err := n.SendCMD(timeoutCtx, nodeID, &rpc.CMDReq{
		Cmd:  rpc.CMDType_ForwardRecvPacket,
		Data: data,
	})
	cancel()
	if err != nil {
		return err
	}
	if resp.Status != rpc.Status_Success {
		return errors.New("转发发送包返回结果失败！")
	}
	return nil
}

// GetSubscribers 获取订阅者
func (n *NodeRemoteCall) GetSubscribers(channelID string, channelType uint8, nodeID int32) ([]string, error) {
	data, _ := proto.Marshal(&rpc.GetSubscribersReq{
		ChannelID:   channelID,
		ChannelType: int32(channelType),
	})
	timeoutCtx, cancel := context.WithTimeout(context.Background(), n.callTimeout)
	resp, err := n.SendCMD(timeoutCtx, nodeID, &rpc.CMDReq{
		Cmd:  rpc.CMDType_GetSubscribers,
		Data: data,
	})
	cancel()
	if err != nil {
		return nil, err
	}
	if resp.Status != rpc.Status_Success {
		return nil, errors.New("转发获取订阅者包返回结果失败！")
	}
	subscribersResp := &rpc.GetSubscribersResp{}
	err = proto.Unmarshal(resp.Data, subscribersResp)
	if err != nil {
		return nil, err
	}
	return subscribersResp.Subscribers, nil
}
