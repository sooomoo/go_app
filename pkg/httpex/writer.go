package httpex

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 自定义响应写入器
type BodyWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func NewBodyWriter(w gin.ResponseWriter, buf *bytes.Buffer) *BodyWriter {
	return &BodyWriter{
		ResponseWriter: w,
		buf:            buf,
	}
}

func (g *BodyWriter) GetBytes() []byte {
	return g.buf.Bytes()
}

func (g *BodyWriter) WriteString(s string) (int, error) {
	g.Header().Del("Content-Length")
	return g.buf.WriteString(s)
}

func (g *BodyWriter) Write(data []byte) (int, error) {
	g.Header().Del("Content-Length")
	return g.buf.Write(data)
}

func (g *BodyWriter) Flush() {
	g.ResponseWriter.Flush()
}

func (g *BodyWriter) WriteHeader(code int) {
	g.Header().Del("Content-Length")
	g.ResponseWriter.WriteHeader(code)
}

func (g *BodyWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := g.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("the ResponseWriter doesn't support the Hijacker interface")
	}
	return hijacker.Hijack()
}
