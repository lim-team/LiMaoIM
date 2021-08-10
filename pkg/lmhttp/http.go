package lmhttp

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/sendgrid/rest"
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

// ForwardWithBody 转发请求
func (c *Context) ForwardWithBody(url string, body []byte) {
	fmt.Println("url->", url)
	fmt.Println("body->", string(body))
	queryMap := map[string]string{}
	values := c.Request.URL.Query()
	if values != nil {
		for key, value := range values {
			queryMap[key] = value[0]
		}
	}
	req := rest.Request{
		Method:      rest.Method(strings.ToUpper(c.Request.Method)),
		BaseURL:     url,
		Headers:     c.CopyRequestHeader(c.Request),
		Body:        body,
		QueryParams: queryMap,
	}

	resp, err := rest.API(req)
	if err != nil {
		c.ResponseError(err)
		return
	}

	fmt.Println("result-code->", resp.StatusCode)
	fmt.Println("result-body->", resp.Body)

	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write([]byte(resp.Body))
}

// Forward 转发请求
func (c *Context) Forward(url string) {
	bodyBytes, _ := ioutil.ReadAll(c.Request.Body)
	c.ForwardWithBody(url, bodyBytes)
}

// CopyRequestHeader 复制request的header参数
func (c *Context) CopyRequestHeader(request *http.Request) map[string]string {
	headerMap := map[string]string{}
	for key, values := range request.Header {
		if len(values) > 0 {
			headerMap[key] = values[0]
		}
	}
	return headerMap
}

// HandlerFunc HandlerFunc
type HandlerFunc func(c *Context)

// LMHttpHandler LMHttpHandler
func (l *LMHttp) LMHttpHandler(handlerFunc HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		hc := l.pool.Get().(*Context)
		hc.reset()
		hc.Context = c
		handlerFunc(hc)
		l.pool.Put(hc)
	}
}

// Run Run
func (l *LMHttp) Run(addr ...string) error {
	return l.r.Run(addr...)
}

// POST POST
func (l *LMHttp) POST(relativePath string, handlers ...HandlerFunc) {
	l.r.POST(relativePath, l.handlersToGinHandleFunc(handlers)...)
}

// GET GET
func (l *LMHttp) GET(relativePath string, handlers ...HandlerFunc) {
	l.r.GET(relativePath, l.handlersToGinHandleFunc(handlers)...)
}

// DELETE DELETE
func (l *LMHttp) DELETE(relativePath string, handlers ...HandlerFunc) {
	l.r.DELETE(relativePath, l.handlersToGinHandleFunc(handlers)...)
}

func (l *LMHttp) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	l.r.ServeHTTP(w, req)
}

// Group Group
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
