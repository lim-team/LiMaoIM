package db

import (
	"errors"
	"os"
)

var (
	// ErrorSegmentReadEnd ErrorSegmentReadEnd
	ErrorSegmentReadEnd = errors.New("Segment file reading end")
	// ErrorReadFinished ErrorReadFinished
	ErrorReadFinished = errors.New("message bakcup file read finished")
)

const (
	fileFormat  = "%020d%s"
	logSuffix   = ".log"
	indexSuffix = ".index"
)

// FileDefaultMode FileDefaultMode
const FileDefaultMode os.FileMode = 0755

var (
	// ErrorNotData ErrorNotData
	ErrorNotData = errors.New("no data")

	// MagicNumber MagicNumber
	MagicNumber = [2]byte{0x15, 0x16} // lm
	// EndMagicNumber EndMagicNumber
	EndMagicNumber = [1]byte{0x3}
	// LogVersion log version
	LogVersion = [1]byte{0x01}
	// SnapshotMagicNumber SnapshotMagicNumber
	SnapshotMagicNumber = [2]byte{0xb, 0xa} // ba
	// EndSnapshotMagicNumber EndSnapshotMagicNumber
	EndSnapshotMagicNumber = [1]byte{0xf}
	// BackupSlotMagicNumber BackupSlotMagicNumber
	BackupSlotMagicNumber = [2]byte{0xc, 0xd}
	// BackupMagicNumber BackupMagicNumber
	BackupMagicNumber = []byte("---backup start ---")
	// BackupMagicEndNumber BackupMagicEndNumber
)

const (
	// OffsetSize OffsetSize
	OffsetSize = 8
	// LogDataLenSize LogDataLenSize
	LogDataLenSize = 4
	// AppliIndexSize AppliIndexSize
	AppliIndexSize = 8
	// LogMaxSize  Maximum size of a single log data
	LogMaxSize = 1024 * 1024
)
