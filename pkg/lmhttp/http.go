package lmhttp

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

type LMHttp struct {
	r    *gin.Engine
	pool sync.Pool
}

func New() *LMHttp {
	l := &LMHttp{
		r:    gin.Default(),
		pool: sync.Pool{},
	}
	l.pool.New = func() interface{} {
		return allocateContext()
	}
	return l
}

// GetGinRoute GetGinRoute
func (l *LMHttp) GetGinRoute() *gin.Engine {
	return l.r
}

// Static Static
func (l *LMHttp) Static(relativePath string, root string) {
	l.r.Static(relativePath, root)
}
func allocateContext() *Context {
	return &Context{Context: nil}
}

type Context struct {
	*gin.Context
}

func (c *Context) reset() {
	c.Context = nil
}

// ResponseError ResponseError
func (c *Context) ResponseError(err error) {
	c.JSON(http.StatusBadRequest, gin.H{
		"msg":    err.Error(),
		"status": http.StatusBadRequest,
	})
}

// ResponseOK 返回正确
func (c *Context) ResponseOK() {
	c.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
	})
}

// ResponseOKWithData 返回正确并并携带数据
func (c *Context) ResponseOKWithData(data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"data":   data,
	})
}

// ResponseData 返回状态和数据
func (c *Context) ResponseData(status int, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"status": status,
		"data":   data,
	})
}

// HandlerFunc
type HandlerFunc func(c *Context)

func (l *LMHttp) LMHttpHandler(handlerFunc HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		hc := l.pool.Get().(*Context)
		hc.reset()
		hc.Context = c
		handlerFunc(hc)
		l.pool.Put(hc)
	}
}

func (l *LMHttp) Run(addr ...string) error {
	return l.r.Run(addr...)
}

func (l *LMHttp) POST(relativePath string, handlers ...HandlerFunc) {
	l.r.POST(relativePath, l.handlersToGinHandleFunc(handlers)...)
}
func (l *LMHttp) GET(relativePath string, handlers ...HandlerFunc) {
	l.r.GET(relativePath, l.handlersToGinHandleFunc(handlers)...)
}

func (l *LMHttp) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	l.r.ServeHTTP(w, req)
}
func (l *LMHttp) Group(relativePath string, handlers ...HandlerFunc) {
	l.r.Group(relativePath, l.handlersToGinHandleFunc(handlers)...)
}

func (l *LMHttp) handlersToGinHandleFunc(handlers []HandlerFunc) []gin.HandlerFunc {
	newHandlers := make([]gin.HandlerFunc, 0, len(handlers))
	if handlers != nil {
		for _, handler := range handlers {
			newHandlers = append(newHandlers, l.LMHttpHandler(handler))
		}
	}
	return newHandlers
}
