package middleware

import (
	"errors"
	"goapp/internal/app"
	"goapp/internal/app/services/crypto"
	"goapp/internal/app/services/headers"
	"io"
	"net/http"
	"strconv"
	"strings"

	"goapp/pkg/httpex"

	"github.com/gin-gonic/gin"
)

func CryptoMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get请求以及指定了不需要加密的路径放行
		cryptoEnabled := app.GetGlobal().GetAuthConfig().EnableCrypto
		if !isPathNeedCrypto(c.Request.URL.Path) || !cryptoEnabled {
			c.Next()
			return
		}

		keys := headers.GetClientKeys(c)
		if keys == nil {
			c.AbortWithError(400, errors.New("bad ckeys"))
			return
		}

		// Get 请求不需要解密
		if c.Request.Method != http.MethodGet && c.Request.ContentLength > 0 {
			// 解密
			contentType := c.GetHeader(headers.HeaderContentType)
			if !strings.HasPrefix(contentType, httpex.ContentTypeEncrypted) {
				c.AbortWithStatus(400)
				return
			}

			reqBody, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatus(500)
				return
			}

			reqBody, err = crypto.Decrypt(keys.ShareKey, reqBody)
			if err != nil {
				c.AbortWithError(400, errors.New("decrypt fail"))
				return
			}

			buf := bufferPool.Get()
			defer bufferPool.Put(buf)
			buf.Write(reqBody)
			c.Request.Body = io.NopCloser(buf)
			c.Request.ContentLength = int64(len(reqBody))
			c.Request.Header.Set(headers.HeaderContentType, getDecryptContentType(c.Request.URL.Path))
		}

		// 代理响应写入器
		respBuf := bufferPool.Get()
		defer bufferPool.Put(respBuf)
		bodyWriter := httpex.NewBodyWriter(c.Writer, respBuf)
		c.Writer = bodyWriter

		c.Next()
		if c.IsAborted() {
			return
		}

		// 加密
		if c.Writer.Status() < 200 || c.Writer.Status() >= 300 {
			return
		}

		responseBody := bodyWriter.GetBytes()
		contentType := c.Writer.Header().Get(headers.HeaderContentType)
		// 仅加密 json 和文本响应
		if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/plain") {
			respBody, err := crypto.Encrypt(keys.ShareKey, responseBody)
			if err != nil {
				c.AbortWithError(500, errors.New("encrypt fail"))
				return
			}

			c.Header(headers.HeaderRawType, contentType)
			c.Header(headers.HeaderContentType, httpex.ContentTypeEncrypted)
			c.Header(headers.HeaderContentLength, strconv.Itoa(len(respBody)))
			bodyWriter.ResponseWriter.WriteString(respBody)
		} else {
			c.Header(headers.HeaderContentLength, strconv.Itoa(len(responseBody)))
			bodyWriter.ResponseWriter.Write(responseBody)
		}

		bodyWriter.ResponseWriter.WriteHeader(c.Writer.Status())
	}
}

func isPathNeedCrypto(path string) bool {
	for _, p := range app.GetGlobal().GetAuthConfig().PathsNotCrypt {
		if strings.Contains(p, "*") || strings.EqualFold(p, path) {
			return false
		}
	}
	for _, p := range app.GetGlobal().GetAuthConfig().PathsNeedCrypt {
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
	return httpex.ContentTypeJson
}
