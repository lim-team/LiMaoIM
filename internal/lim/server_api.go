package lim

import (
	_ "net/http/pprof" // pprof

	"github.com/gin-contrib/pprof"
	"github.com/lim-team/LiMaoIM/pkg/lmhttp"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"go.uber.org/zap"
)

// APIServer ApiServer
type APIServer struct {
	r    *lmhttp.LMHttp
	addr string
	l    *LiMao
	limlog.Log
}

// NewAPIServer new一个api server
func NewAPIServer(l *LiMao) *APIServer {
	r := lmhttp.New()
	//pprof.Register(r.GetGinRoute(), "dev/pprof")
	r.Static("/api", "./configs/swagger")
	if l.opts.Mode == TestMode {
		pprof.Register(r.GetGinRoute()) // 性能
	}
	hs := &APIServer{
		r:    r,
		addr: l.opts.HTTPAddr,
		l:    l,
		Log:  limlog.NewLIMLog("APIServer"),
	}

	return hs
}

// Start 开始
func (s *APIServer) Start() {
	// runtime.SetMutexProfileFraction(1) // 开启对锁调用的跟踪
	// runtime.SetBlockProfileRate(1)     // 开启对阻塞操作的跟踪
	s.setRoutes()
	go func() {
		err := s.r.Run(s.addr) // listen and serve
		if err != nil {
			panic(err)
		}
	}()
	s.Info("服务开启", zap.String("addr", s.l.opts.HTTPAddr))
}

func (s *APIServer) setRoutes() {
	// 用户相关API
	u := NewUserAPI(s.l)
	u.Route(s.r)
	// 消息相关API
	message := NewMessageAPI(s.l)
	message.Route(s.r)
	// 频道相关API
	channel := NewChannelAPI(s.l)
	channel.Route(s.r)
	// 最近会话API
	conversation := NewConversationAPI(s.l)
	conversation.Route(s.r)
}

// Stop 停止服务
func (s *APIServer) Stop() {
}
