package middlewares

import (
	"bytes"
	"errors"
	"fmt"
	"goapp/internal/app/config"
	"goapp/internal/app/global"
	"goapp/internal/app/services"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

// Authenticator 需要处理以下内容：
// 1. 验证请求是否合法有效
// 2. 验证请求是否被篡改
// 3. 验证请求是否被重复使用
// 4. 如果请求已经加密，需要解密，并在处理业务逻辑之后加密响应内容
type Authenticator struct {
	config     *config.AuthenticatorConfig
	bufferPool sync.Pool
	authSvr    *services.AuthService
}

func (d *Authenticator) getBuffer() *bytes.Buffer {
	buf := d.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func (d *Authenticator) isPathEncrypted(path string) bool {
	for _, p := range d.config.PathsNotCrypt {
		if strings.Contains(p, "*") || strings.EqualFold(p, path) {
			return false
		}
	}
	for _, p := range d.config.PathsNeedCrypt {
		if strings.Contains(p, "*") {
			return true
		}
		if strings.EqualFold(p, path) {
			return true
		}
	}
	return false
}

func (d *Authenticator) getDecryptContentType(path string) string {
	return niu.ContentTypeJson
}

func (d *Authenticator) isPathNeedAuth(path string) bool {
	for _, p := range d.config.PathsNotAuth {
		if strings.EqualFold(p, path) {
			return false
		}
	}
	for _, p := range d.config.PathsNeedAuth {
		if strings.Contains(p, "*") {
			return true
		}
		if strings.EqualFold(p, path) {
			return true
		}
	}
	return false
}

// 用于生成待签名的内容
func (d *Authenticator) stringfySignData(params map[string]string) []byte {
	// 对参数名进行排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 拼接参数
	b := strings.Builder{}
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("%s=%s\n", k, params[k]))
	}
	return []byte(b.String())
}

func (d *Authenticator) verifyToken(c *gin.Context) {
	tokenString := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	if d.isPathNeedAuth(c.Request.URL.Path) {
		if len(tokenString) == 0 {
			c.AbortWithStatus(401)
			return
		}
		revoked, err := d.authSvr.IsTokenRevoked(c, tokenString)
		if err != nil {
			c.AbortWithError(500, errors.New("check token revoke fail"))
			return
		}
		if revoked {
			c.AbortWithStatus(401)
			return
		}
		// 解析Token
		claims, err := d.authSvr.ParseToken(tokenString)
		if err != nil {
			c.AbortWithError(401, errors.New("invalid token"))
			return
		}
		c.Set(services.KeyClaims, claims)
	} else if len(tokenString) > 0 {
		revoked, _ := d.authSvr.IsTokenRevoked(c, tokenString)
		if !revoked {
			// 解析Token
			claims, err := d.authSvr.ParseToken(tokenString)
			if err == nil {
				// 忽略错误
				c.Set(services.KeyClaims, claims)
			}
		}
	}
}

func (d *Authenticator) checkModified(c *gin.Context, nonce, timestamp, platform, signature string) ([]byte, niu.Signer) {
	reqBody := make([]byte, 0, 0)
	if c.Request.Body != nil {
		var err error
		reqBody, err = io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatus(500)
			return nil, nil
		}
	}

	// 3. 先验证签名是否正确，在根据需要解密请求体
	signer, err := d.authSvr.GetSigner(c)
	if err != nil {
		c.AbortWithError(500, err)
		return nil, nil
	}
	signdata := d.stringfySignData(map[string]string{
		"nonce":     nonce,
		"timestamp": timestamp,
		"platform":  platform,
		"method":    c.Request.Method,
		"path":      c.Request.URL.Path,
		"query":     c.Request.URL.RawQuery,
		"body":      string(reqBody),
	})
	if !signer.Verify(signdata, []byte(signature)) {
		c.AbortWithError(400, errors.New("invalid signature"))
		return nil, signer
	}

	return reqBody, signer
}

func (d *Authenticator) replaceRequestBody(c *gin.Context, reqBody []byte) niu.Cryptor {
	if len(reqBody) == 0 {
		return nil
	}

	var cryptor niu.Cryptor = nil

	if d.isPathEncrypted(c.Request.URL.Path) {
		contentType := c.GetHeader("Content-Type")
		if !strings.EqualFold(contentType, niu.ContentTypeEncrypted) {
			c.AbortWithStatus(400)
			return nil
		}
		var err error
		cryptor, err = d.authSvr.GetCryptor(c)
		if err != nil {
			c.AbortWithError(500, err)
			return nil
		}
		// 解密
		reqBody, err = cryptor.Decrypt(reqBody)
		if err != nil {
			c.AbortWithError(400, errors.New("decrypt fail"))
			return nil
		}

		c.Request.Header.Set("Content-Type", d.getDecryptContentType(c.Request.URL.Path))
	}

	buf := d.getBuffer()
	buf.Write(reqBody)
	c.Request.Body = io.NopCloser(buf)
	c.Request.ContentLength = int64(len(reqBody))

	return cryptor
}

func (d *Authenticator) replaceReponseBody(c *gin.Context, respBody []byte, signer niu.Signer, cryptor niu.Cryptor, nonce, timestamp, platform string) (contentType string, body, signature []byte) {
	respContentType := c.Writer.Header().Get("Content-Type")
	if d.isPathEncrypted(c.Request.URL.Path) {
		if cryptor == nil {
			panic("cryptor is nil")
		}
		var err error
		respBody, err = cryptor.Encrypt(respBody)
		if err != nil {
			c.AbortWithError(500, errors.New("encrypt fail"))
			return "", nil, nil
		}
		respContentType = niu.ContentTypeEncrypted
	}

	// 9. 生成响应签名
	respSignData := d.stringfySignData(map[string]string{
		"nonce":     nonce,
		"platform":  platform,
		"timestamp": timestamp,
		"method":    c.Request.Method,
		"path":      c.Request.RequestURI,
		"query":     c.Request.URL.RawQuery,
		"body":      string(respBody),
	})
	respSignature, err := signer.Sign(respSignData)
	if err != nil {
		c.AbortWithError(500, errors.New("sign fail"))
		return "", nil, nil
	}

	return respContentType, respBody, respSignature
}

// 自定义响应写入器
type bodyWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w bodyWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func AuthenticateMiddleware() gin.HandlerFunc {
	d := &Authenticator{
		config: &global.AppConfig.Authenticator,
		bufferPool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, 1024))
			},
		},
		authSvr: services.NewAuthService(),
	}

	return func(c *gin.Context) {
		nonce := strings.TrimSpace(c.GetHeader("X-Nonce"))
		timestampStr := strings.TrimSpace(c.GetHeader("X-Timestamp"))
		platform := strings.TrimSpace(c.GetHeader("X-Platform"))
		signature := c.GetHeader("X-Signature")
		if len(nonce) == 0 || len(timestampStr) == 0 || len(signature) == 0 || !niu.IsPlatformStringValid(platform) {
			c.AbortWithStatus(400)
			return
		}

		// 1. 验证请求是否是重放请求
		if d.authSvr.IsReplayRequest(c, nonce, timestampStr) {
			c.AbortWithError(400, errors.New("repeat request"))
			return
		}

		// 2. 解码Token（如果有），第三步会用到解码之后的token
		d.verifyToken(c)
		if c.IsAborted() {
			return
		}

		// 3. 验证请求是否被篡改
		reqBody, signer := d.checkModified(c, nonce, timestampStr, platform, signature)
		if c.IsAborted() {
			return
		}

		// 4. 修改并重置请求体: 需验证请求是否加密，如果加密，则解密
		cryptor := d.replaceRequestBody(c, reqBody)
		if c.IsAborted() {
			return
		}

		// 5. 代理响应写入器
		respBuf := d.getBuffer()
		bodyWriter := &bodyWriter{ResponseWriter: c.Writer, buf: respBuf}
		c.Writer = bodyWriter

		// 6. 处理业务逻辑
		c.Next()
		if c.IsAborted() {
			return
		}

		// 7. 如果需要加密，则加密响应内容
		responseBody := bodyWriter.buf.Bytes()
		respTimestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		respNonce := niu.NewUUIDWithoutDash()
		respContentType, responseBody, respSignature := d.replaceReponseBody(c, responseBody, signer, cryptor, respNonce, respTimestamp, platform)
		if c.IsAborted() {
			return
		}
		// 8. 写入响应头
		c.Header("X-Signature", respTimestamp)
		c.Header("X-Nonce", respNonce)
		c.Header("X-Timestamp", string(respSignature))
		c.Header("Content-Type", respContentType)
		c.Header("Content-Length", strconv.Itoa(len(responseBody)))
		c.Writer.Write(responseBody)
	}
}
