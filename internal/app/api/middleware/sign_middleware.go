package middleware

import (
	"errors"
	"goapp/internal/app/service"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

func SignMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce := strings.TrimSpace(c.GetHeader("X-Nonce"))
		timestampStr := strings.TrimSpace(c.GetHeader("X-Timestamp"))
		platform := strings.TrimSpace(c.GetHeader("X-Platform"))
		signature := strings.TrimSpace(c.GetHeader("X-Signature"))
		sessionId := strings.TrimSpace(c.GetHeader("X-Session"))

		// 1. 验证请求是否签名是否正确
		svc := service.NewAuthService()
		dataToVerify := map[string]string{
			"session":   sessionId,
			"nonce":     nonce,
			"timestamp": timestampStr,
			"platform":  platform,
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"query":     c.Request.URL.RawQuery,
		}
		if c.Request.Method != "GET" {
			reqBody, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatus(500)
				return
			}
			dataToVerify["body"] = string(reqBody)

			buf := bufferPool.Get()
			defer bufferPool.Put(buf)
			buf.Write(reqBody)
			c.Request.Body = io.NopCloser(buf)
			c.Request.ContentLength = int64(len(reqBody))
		}

		// 验证签名是否正确
		verified, err := svc.SignVerify(c, dataToVerify, signature)
		if err != nil {
			c.AbortWithError(500, errors.New("verify signature fail"))
			return
		}
		if !verified {
			c.AbortWithError(400, errors.New("invalid signature"))
			return
		}

		// 代理响应写入器
		respBuf := bufferPool.Get()
		defer bufferPool.Put(respBuf)
		bodyWriter := &bodyWriter{ResponseWriter: c.Writer, buf: respBuf}
		c.Writer = bodyWriter

		c.Next()
		if c.IsAborted() {
			return
		}

		// 签名响应体
		responseBody := bodyWriter.buf.Bytes()
		respTimestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		respNonce := niu.NewUUIDWithoutDash()
		dataToSign := map[string]string{
			"session":   sessionId,
			"nonce":     respNonce,
			"platform":  platform,
			"timestamp": respTimestamp,
			"method":    c.Request.Method,
			"path":      c.Request.RequestURI,
			"query":     c.Request.URL.RawQuery,
			"body":      string(responseBody),
		}

		// 生成响应签名
		respSignature, err := svc.Sign(c, dataToSign)
		if err != nil {
			c.AbortWithError(500, errors.New("sign fail"))
			return
		}

		// 8. 写入响应头
		c.Header("X-Signature", respTimestamp)
		c.Header("X-Nonce", respNonce)
		c.Header("X-Timestamp", respSignature)
	}
}
