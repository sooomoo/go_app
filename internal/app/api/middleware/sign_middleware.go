package middleware

import (
	"errors"
	"goapp/internal/pkg/crypto"
	"io"
	"net/url"
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

		keys := getClientKeys(c)
		if keys == nil {
			c.AbortWithError(400, errors.New("bad skeys"))
			return
		}

		// 1. 验证请求是否签名是否正确
		dataToVerify := map[string]string{
			"session":   sessionId,
			"nonce":     nonce,
			"timestamp": timestampStr,
			"platform":  platform,
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"query":     string(crypto.StringfyMap(convertValuesToMap(c.Request.URL.Query()))),
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
		verified, err := crypto.VerifySignMap(keys.SignPubKey, dataToVerify, signature)
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

		if c.Writer.Status() < 200 || c.Writer.Status() >= 300 {
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
		respSignature, err := crypto.SignMap(dataToSign)
		if err != nil {
			c.AbortWithError(500, errors.New("sign fail"))
			return
		}

		// 8. 写入响应头
		c.Header("x-timestamp", respTimestamp)
		c.Header("x-nonce", respNonce)
		c.Header("x-signature", respSignature)
		c.Header("Content-Length", strconv.Itoa(len(responseBody)))

		bodyWriter.ResponseWriter.Write(responseBody)
		bodyWriter.ResponseWriter.WriteHeader(c.Writer.Status())
	}
}

func convertValuesToMap(values url.Values) map[string]string {
	mp := make(map[string]string)
	for k, v := range values {
		mp[k] = strings.Join(v, ",")
	}
	return mp
}
