package middleware

import (
	"bytes"
	"errors"
	"goapp/internal/pkg/crypto"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

var bufferPool *niu.ByteBufferPool = niu.NewByteBufferPool(0, 1024)

// 自定义响应写入器
type bodyWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w bodyWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

const (
	KeyClaims     = "claims"
	KeyClientKeys = "client_keys"
)

type ClientKeys struct {
	SignPubKey []byte
	BoxPubKey  []byte
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

	nonce := raw[:17]
	remain := raw[17:]
	for i := range remain {
		elem := remain[i]
		remain[i] = elem ^ nonce[i%17]
	}
	signPubKey := remain[7:39]
	boxPubKey := remain[39:71]
	ctx.Set(KeyClientKeys, &ClientKeys{signPubKey, boxPubKey})
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
