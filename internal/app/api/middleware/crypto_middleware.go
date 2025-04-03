package middleware

import (
	"errors"
	"goapp/internal/app/global"
	"goapp/internal/app/service"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

func CryptoMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get请求以及指定了不需要加密的路径放行
		if c.Request.Method == "GET" || !isPathNeedCrypto(c.Request.URL.Path) {
			c.Next()
			return
		}

		// 解密
		svc := service.NewAuthService()
		contentType := c.GetHeader("Content-Type")
		if !strings.EqualFold(contentType, niu.ContentTypeEncrypted) {
			c.AbortWithStatus(400)
			return
		}

		reqBody, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatus(500)
			return
		}

		reqBody, err = svc.Decrypt(c, reqBody)
		if err != nil {
			c.AbortWithError(400, errors.New("decrypt fail"))
			return
		}

		buf := bufferPool.Get()
		defer bufferPool.Put(buf)
		buf.Write(reqBody)
		c.Request.Body = io.NopCloser(buf)
		c.Request.ContentLength = int64(len(reqBody))
		c.Request.Header.Set("Content-Type", getDecryptContentType(c.Request.URL.Path))

		// 代理响应写入器
		respBuf := bufferPool.Get()
		defer bufferPool.Put(respBuf)
		bodyWriter := &bodyWriter{ResponseWriter: c.Writer, buf: respBuf}
		c.Writer = bodyWriter

		c.Next()
		if c.IsAborted() {
			return
		}

		// 加密
		responseBody := bodyWriter.buf.Bytes()
		respBody, err := svc.Encrypt(c, responseBody)
		if err != nil {
			c.AbortWithError(500, errors.New("encrypt fail"))
			return
		}

		c.Header("Content-Type", niu.ContentTypeEncrypted)
		c.Header("Content-Length", strconv.Itoa(len(respBody)))
		bodyWriter.ResponseWriter.Write(respBody)
	}
}

func isPathNeedCrypto(path string) bool {
	for _, p := range global.AppConfig.Authenticator.PathsNotCrypt {
		if strings.Contains(p, "*") || strings.EqualFold(p, path) {
			return false
		}
	}
	for _, p := range global.AppConfig.Authenticator.PathsNeedCrypt {
		if strings.Contains(p, "*") {
			return true
		}
		if strings.EqualFold(p, path) {
			return true
		}
	}
	return false
}

func getDecryptContentType(_ string) string {
	return niu.ContentTypeJson
}
