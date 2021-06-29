package lim

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLiMaoStartAndStop(t *testing.T) {
	l := NewTestLiMao()
	err := l.Start()
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 10)
	err = l.Stop()
	assert.NoError(t, err)
}
