package middleware

import (
	"bytes"
	"errors"
	"goapp/internal/app/config"
	"goapp/internal/app/global"
	"goapp/internal/app/service"
	"io"
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
	authSvr    *service.AuthService
}

func (d *Authenticator) getBuffer() *bytes.Buffer {
	buf := d.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func (d *Authenticator) isPathNeedCrypto(path string) bool {
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

func (d *Authenticator) getDecryptContentType(_ string) string {
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

func (d *Authenticator) verifyToken(c *gin.Context) {
	tokens := d.authSvr.GetAuthorizationHeader(c)
	isTokenValid := len(tokens) == 2 && tokens[0] == "Bearer" || len(tokens[1]) > 0
	if d.isPathNeedAuth(c.Request.URL.Path) {
		if !isTokenValid {
			c.AbortWithStatus(401)
			return
		}
		tokenString := tokens[1]

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

		// 刷新Token时，此处的类型为 r
		allowTokenTyep := "a"
		if strings.EqualFold(c.Request.URL.Path, d.config.RefreshTokenPath) {
			allowTokenTyep = "r"
		}
		if claims.Type != allowTokenTyep {
			c.AbortWithError(401, errors.New("invalid token type"))
			return
		}

		d.authSvr.SaveClaims(c, claims)
	} else if isTokenValid {
		tokenString := tokens[1]
		revoked, _ := d.authSvr.IsTokenRevoked(c, tokenString)
		if !revoked {
			// 解析Token
			claims, err := d.authSvr.ParseToken(tokenString)
			if err == nil {
				// 忽略错误
				d.authSvr.SaveClaims(c, claims)
			}
		}
	}
}

func (d *Authenticator) checkModified(c *gin.Context, nonce, timestamp, platform, signature string) []byte {
	reqBody := make([]byte, 0)
	if c.Request.Body != nil {
		var err error
		reqBody, err = io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatus(500)
			return nil
		}
	}

	// 验证签名是否正确
	verified, err := d.authSvr.SignVerify(c, map[string]string{
		"nonce":     nonce,
		"timestamp": timestamp,
		"platform":  platform,
		"method":    c.Request.Method,
		"path":      c.Request.URL.Path,
		"query":     c.Request.URL.RawQuery,
		"body":      string(reqBody),
	}, signature)
	if err != nil {
		c.AbortWithError(500, errors.New("verify signature fail"))
		return nil
	}
	if !verified {
		c.AbortWithError(400, errors.New("invalid signature"))
		return nil
	}

	return reqBody
}

func (d *Authenticator) replaceRequestBody(c *gin.Context, reqBody []byte) {
	if len(reqBody) == 0 || c.Request.Method == "GET" {
		return
	}

	if d.isPathNeedCrypto(c.Request.URL.Path) {
		contentType := c.GetHeader("Content-Type")
		if !strings.EqualFold(contentType, niu.ContentTypeEncrypted) {
			c.AbortWithStatus(400)
			return
		}
		// 解密
		var err error
		reqBody, err = d.authSvr.Decrypt(c, reqBody)
		if err != nil {
			c.AbortWithError(400, errors.New("decrypt fail"))
			return
		}

		c.Request.Header.Set("Content-Type", d.getDecryptContentType(c.Request.URL.Path))
	}

	buf := d.getBuffer()
	buf.Write(reqBody)
	c.Request.Body = io.NopCloser(buf)
	c.Request.ContentLength = int64(len(reqBody))
}

func (d *Authenticator) replaceReponseBody(c *gin.Context, respBody []byte, nonce, timestamp, platform string) (contentType string, body []byte, signature string) {
	respContentType := c.Writer.Header().Get("Content-Type")
	if d.isPathNeedCrypto(c.Request.URL.Path) {
		var err error
		respBody, err = d.authSvr.Encrypt(c, respBody)
		if err != nil {
			c.AbortWithError(500, errors.New("encrypt fail"))
			return "", nil, ""
		}
		respContentType = niu.ContentTypeEncrypted
	}

	// 9. 生成响应签名

	respSignature, err := d.authSvr.Sign(c, map[string]string{
		"nonce":     nonce,
		"platform":  platform,
		"timestamp": timestamp,
		"method":    c.Request.Method,
		"path":      c.Request.RequestURI,
		"query":     c.Request.URL.RawQuery,
		"body":      string(respBody),
	})
	if err != nil {
		c.AbortWithError(500, errors.New("sign fail"))
		return "", nil, ""
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
		authSvr: service.NewAuthService(),
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
		reqBody := d.checkModified(c, nonce, timestampStr, platform, signature)
		if c.IsAborted() {
			return
		}

		// 4. 修改并重置请求体: 需验证请求是否加密，如果加密，则解密
		d.replaceRequestBody(c, reqBody)
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
		respContentType, responseBody, respSignature := d.replaceReponseBody(c, responseBody, respNonce, respTimestamp, platform)
		if c.IsAborted() {
			return
		}
		// 8. 写入响应头
		c.Header("X-Signature", respTimestamp)
		c.Header("X-Nonce", respNonce)
		c.Header("X-Timestamp", respSignature)
		c.Header("Content-Type", respContentType)
		c.Header("Content-Length", strconv.Itoa(len(responseBody)))
		bodyWriter.ResponseWriter.Write(responseBody)
	}
}
