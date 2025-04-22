package middleware

import (
	"bufio"
	"bytes"
	"errors"
	"goapp/internal/app/global"
	"goapp/internal/app/service"
	"goapp/internal/pkg/crypto"
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

func parseAndStoreClientKeys(ctx *gin.Context, sessionId string) *service.SessionClientKeys {
	raw, err := crypto.Base64Decode(sessionId)
	if err != nil {
		ctx.AbortWithError(400, errors.New("bad session id"))
		return nil
	}

	if len(raw) != 88 {
		ctx.AbortWithError(400, errors.New("bad session id"))
		return nil
	}

	for i := 17; i < len(raw); i++ {
		elem := raw[i]
		raw[i] = elem ^ raw[i%17]
	}

	signPubKey := raw[24:56]
	boxPubKey := raw[56:]
	if global.AppConfig.Authenticator.EnableCrypto {
		shareKey, err := crypto.NegotiateShareKey(boxPubKey, global.AppConfig.Authenticator.BoxKeyPair.PrivateKey)
		if err != nil {
			ctx.AbortWithError(400, errors.New("negotiate fail"))
			return nil
		}

		skeys := &service.SessionClientKeys{
			SignPubKey: signPubKey,
			BoxPubKey:  boxPubKey,
			ShareKey:   shareKey,
		}
		ctx.Set(service.KeyClientKeys, skeys)
		return skeys
	} else {
		skeys := &service.SessionClientKeys{
			SignPubKey: signPubKey,
			BoxPubKey:  boxPubKey,
			ShareKey:   nil,
		}
		ctx.Set(service.KeyClientKeys, skeys)
		return skeys
	}
}

func getClientKeys(ctx *gin.Context) *service.SessionClientKeys {
	val, exist := ctx.Get(service.KeyClientKeys)
	if !exist {
		return nil
	}
	keys, ok := val.(*service.SessionClientKeys)
	if !ok {
		return nil
	}
	return keys
}
