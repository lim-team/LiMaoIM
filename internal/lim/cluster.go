package lim

import (
	"context"

	"github.com/lim-team/LiMaoIM/pkg/lmproxyproto"
)

// IClusterManager IClusterManager
type IClusterManager interface {
	Start() error
	Stop() error
	IsLeader() bool
	GetNode(value string) *lmproxyproto.Node
	GetNodeWithID(id int32) *lmproxyproto.Node
	SyncPropose(ctx context.Context, data []byte) error
	SendCMD(cmd *lmproxyproto.CMD) error
	GetClusterConfig() *lmproxyproto.ClusterConfig
	LeaderUpdated(f func(leaderID uint32))
}
