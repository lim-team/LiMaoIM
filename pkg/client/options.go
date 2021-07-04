package client

import "github.com/lim-team/LiMaoIM/pkg/lmproto"

// Options Options
type Options struct {
	ProtoVersion uint8  // 协议版本
	UID          string // 用户uid
	Token        string // 连接IM的token
	AutoReconn   bool   //是否开启自动重连
}

// NewOptions 创建默认配置
func NewOptions() *Options {
	return &Options{
		ProtoVersion: lmproto.LatestVersion,
		AutoReconn:   false,
	}
}

// Option 参数项
type Option func(*Options) error

// WithProtoVersion 设置协议版本
func WithProtoVersion(version uint8) Option {
	return func(opts *Options) error {
		opts.ProtoVersion = version
		return nil
	}
}

// WithUID 用户UID
func WithUID(uid string) Option {
	return func(opts *Options) error {
		opts.UID = uid
		return nil
	}
}

// WithToken 用户token
func WithToken(token string) Option {
	return func(opts *Options) error {
		opts.Token = token
		return nil
	}
}

// WithAutoReconn WithAutoReconn
func WithAutoReconn(autoReconn bool) Option {
	return func(opts *Options) error {
		opts.AutoReconn = autoReconn
		return nil
	}
}

// SendOptions SendOptions
type SendOptions struct {
	NoPersist bool // 是否不存储 默认 false
	SyncOnce  bool // 是否同步一次（写模式） 默认 false
	Flush     bool // 是否io flush 默认true
	RedDot    bool // 是否显示红点 默认true
}

// NewSendOptions NewSendOptions
func NewSendOptions() *SendOptions {
	return &SendOptions{
		NoPersist: false,
		SyncOnce:  false,
		Flush:     true,
		RedDot:    true,
	}
}

// SendOption 参数项
type SendOption func(*SendOptions) error

// SendOptionWithNoPersist 是否不存储
func SendOptionWithNoPersist(noPersist bool) SendOption {
	return func(opts *SendOptions) error {
		opts.NoPersist = noPersist
		return nil
	}
}

// SendOptionWithSyncOnce 是否只同步一次（写模式）
func SendOptionWithSyncOnce(syncOnce bool) SendOption {
	return func(opts *SendOptions) error {
		opts.SyncOnce = syncOnce
		return nil
	}
}

// SendOptionWithFlush 是否 io flush
func SendOptionWithFlush(flush bool) SendOption {
	return func(opts *SendOptions) error {
		opts.Flush = flush
		return nil
	}
}

// SendOptionWithRedDot 是否显示红点
func SendOptionWithRedDot(redDot bool) SendOption {
	return func(opts *SendOptions) error {
		opts.RedDot = redDot
		return nil
	}
}
