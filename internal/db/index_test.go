package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexNew(t *testing.T) {
	dir, err := ioutil.TempDir("", "commitlog-index")
	require.NoError(t, err)
	path := filepath.Join(dir, "test.index")
	defer os.Remove(path)

	idx := NewIndex(path, 0)

	defer idx.Close()

	fmt.Println(idx.entrySize)
	idx.TruncateEntries(0)
	for i := 0; i < 1000; i++ {
		idx.Append(int64(i+1000), int64(1+i))
	}
	offset, err := idx.Lookup(1400)
	assert.NoError(t, err)

	assert.Equal(t, uint64(1400), offset.Offset)
}
