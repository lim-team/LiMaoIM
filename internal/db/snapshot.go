package db

// Snapshot Snapshot
type Snapshot struct {
	AppliIndex uint64
}

// NewSnapshot NewSnapshot
func NewSnapshot(appliIndex uint64) *Snapshot {
	return &Snapshot{
		AppliIndex: appliIndex,
	}
}
