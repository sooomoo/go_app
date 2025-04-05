package middleware

import (
	"bytes"
	"errors"
	"goapp/internal/app/global"
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
func (w bodyWriter) WriteString(s string) (int, error) {
	return w.buf.WriteString(s)
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
