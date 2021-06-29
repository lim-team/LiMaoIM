package lim

import (
	"io/ioutil"

	"github.com/lim-team/LiMaoIM/pkg/client"
)

const (
	TestUID = "test"
)

// NewTestOptions NewTestOptions
func NewTestOptions() *Options {
	opts := NewOptions()
	opts.Mode = TestMode
	dir, err := ioutil.TempDir("", "limao-test")
	if err != nil {
		panic(err)
	}
	opts.DataDir = dir
	opts.Load()

	return opts
}

// NewTestLiMao NewTestLiMao
func NewTestLiMao(ots ...*Options) *LiMao {
	var opts *Options
	if len(ots) > 0 {
		opts = ots[0]
	} else {
		opts = NewTestOptions()
	}

	l := New(opts)
	return l
}

// MustConnectLiMao MustConnectLiMao
func MustConnectLiMao(l *LiMao, uid string) *client.Client {
	c := client.New(l.lnet.GetTCPServer().GetRealAddr(), client.WithUID(uid))
	err := c.Connect()
	if err != nil {
		panic(err)
	}
	return c
}
