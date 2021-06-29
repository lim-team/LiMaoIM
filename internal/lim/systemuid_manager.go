package lim

import "sync"

// SystemUIDManager System uid management
type SystemUIDManager struct {
	datasource IDatasource
	l          *LiMao
	systemUIDs sync.Map
}

// NewSystemUIDManager NewSystemUIDManager
func NewSystemUIDManager(l *LiMao) *SystemUIDManager {

	return &SystemUIDManager{
		l:          l,
		datasource: NewDatasource(l),
		systemUIDs: sync.Map{},
	}
}

// LoadIfNeed LoadIfNeed
func (s *SystemUIDManager) LoadIfNeed() error {
	if !s.l.opts.HasDatasource() {
		return nil
	}
	var err error
	systemUIDs, err := s.datasource.GetSystemUIDs()
	if err != nil {
		return err
	}
	if len(systemUIDs) > 0 {
		for _, systemUID := range systemUIDs {
			s.systemUIDs.Store(systemUID, true)
		}
	}
	return nil
}

// SystemUID Is it a system account?
func (s *SystemUIDManager) SystemUID(uid string) bool {
	_, ok := s.systemUIDs.Load(uid)
	return ok
}

// AddSystemUID AddSystemUID
func (s *SystemUIDManager) AddSystemUID(uid string) {
	s.systemUIDs.Store(uid, true)
}

// RemoveSystemUID RemoveSystemUID
func (s *SystemUIDManager) RemoveSystemUID(uid string) {
	s.systemUIDs.Delete(uid)
}
