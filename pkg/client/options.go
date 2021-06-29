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
