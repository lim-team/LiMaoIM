package lim

import (
	"io"

	"github.com/lim-team/LiMaoIM/internal/db"
	sm "github.com/lni/dragonboat/v3/statemachine"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"go.uber.org/zap"
)

type stateMachine struct {
	clusterID   uint64
	nodeID      int32
	db          db.DB
	lastApplied uint64
	limlog.Log
	slotCount int
}

func newStateMachine(dataDir string, clusterID uint64, nodeID int32, slotCount int, segmentMaxBytes int64) *stateMachine {
	return &stateMachine{
		clusterID: clusterID,
		nodeID:    nodeID,
		db:        db.NewFileDB(dataDir, segmentMaxBytes, slotCount),
		slotCount: slotCount,
		Log:       limlog.NewLIMLog("stateMachine"),
	}
}

func (s *stateMachine) Open(stopc <-chan struct{}) (uint64, error) {
	err := s.db.Open()
	if err != nil {
		return 0, err
	}
	appliedIndex, err := s.db.GetMetaData()
	if err != nil {
		return 0, err
	}
	s.lastApplied = appliedIndex
	return appliedIndex, nil
}

func (s *stateMachine) Update(entries []sm.Entry) ([]sm.Entry, error) {
	var err error
	for _, e := range entries {
		cmd := &CMD{}
		err = UnmarshalCMD(e.Cmd, cmd)
		if err != nil {
			panic(err)
		}
		if cmd.Type == CMDAppendMessage {
			var message = &db.Message{}
			err = db.UnmarshalMessage(cmd.Param, message)
			if err != nil {
				s.Warn("Unmarshal message fail!", zap.Error(err))
			} else {
				_, err = s.db.AppendMessage(message)
				if err != nil {
					return nil, err
				}
			}

		}
	}

	err = s.db.SaveMetaData(entries[len(entries)-1].Index)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *stateMachine) Lookup(interface{}) (interface{}, error) {
	return nil, nil
}

func (s *stateMachine) Sync() error {
	return nil
}

func (s *stateMachine) PrepareSnapshot() (interface{}, error) {
	return nil, nil
}

func (s *stateMachine) SaveSnapshot(interface{}, io.Writer, <-chan struct{}) error {
	return nil
}

func (s *stateMachine) RecoverFromSnapshot(io.Reader, <-chan struct{}) error {
	return nil
}

func (s *stateMachine) Close() error {
	return s.db.Close()
}
