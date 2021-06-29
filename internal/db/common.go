package db

import (
	"errors"
	"os"
)

var (
	// ErrorSegmentReadEnd ErrorSegmentReadEnd
	ErrorSegmentReadEnd = errors.New("Segment file reading end")
)

const (
	fileFormat  = "%020d%s"
	logSuffix   = ".log"
	indexSuffix = ".index"
)

// FileDefaultMode FileDefaultMode
const FileDefaultMode os.FileMode = 0755
