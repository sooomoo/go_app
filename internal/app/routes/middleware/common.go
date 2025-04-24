package middleware

import (
	"bufio"
	"bytes"
	"errors"
	"goapp/pkg/core"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

var bufferPool *core.ByteBufferPool = core.NewByteBufferPool(0, 1024)

// 自定义响应写入器
type bodyWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (g *bodyWriter) WriteString(s string) (int, error) {
	g.Header().Del("Content-Length")
	return g.buf.WriteString(s)
}

func (g *bodyWriter) Write(data []byte) (int, error) {
	g.Header().Del("Content-Length")
	return g.buf.Write(data)
}

func (g *bodyWriter) Flush() {
	g.ResponseWriter.Flush()
}

func (g *bodyWriter) WriteHeader(code int) {
	g.Header().Del("Content-Length")
	g.ResponseWriter.WriteHeader(code)
}

func (g *bodyWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := g.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("the ResponseWriter doesn't support the Hijacker interface")
	}
	return hijacker.Hijack()
}
