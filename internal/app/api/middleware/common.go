package middleware

import (
	"bufio"
	"bytes"
	"errors"
	"goapp/internal/app/global"
	"goapp/internal/pkg/crypto"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

var bufferPool *niu.ByteBufferPool = niu.NewByteBufferPool(0, 1024)

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

const (
	KeyClaims     = "claims"
	KeyClientKeys = "client_keys"
)

type ClientKeys struct {
	SignPubKey []byte
	BoxPubKey  []byte
	ShareKey   []byte
}

func parseAndStoreClientKeys(ctx *gin.Context, sessionId string) {
	raw, err := crypto.Base64Decode(sessionId)
	if err != nil {
		ctx.AbortWithError(400, errors.New("bad session id"))
		return
	}

	if len(raw) != 88 {
		ctx.AbortWithError(400, errors.New("bad session id"))
		return
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
			return
		}

		ctx.Set(KeyClientKeys, &ClientKeys{signPubKey, boxPubKey, shareKey})
	} else {
		ctx.Set(KeyClientKeys, &ClientKeys{signPubKey, boxPubKey, nil})
	}
}

func getClientKeys(ctx *gin.Context) *ClientKeys {
	val, exist := ctx.Get(KeyClientKeys)
	if !exist {
		return nil
	}
	keys, ok := val.(*ClientKeys)
	if !ok {
		return nil
	}
	return keys
}
