package middleware

import (
	"errors"
	"fmt"
	"goapp/internal/app/service/headers"
	"goapp/internal/pkg/crypto"
	"goapp/pkg/core"
	"goapp/pkg/httpex"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func SignMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := strings.TrimSpace(c.GetHeader(headers.HeaderAuthorization))

		extendData := headers.GetExtendData(c)
		keys := headers.GetClientKeys(c)
		if keys == nil || extendData == nil {
			c.AbortWithError(400, errors.New("bad skeys or extend data"))
			return
		}

		method := strings.ToUpper(c.Request.Method)

		queryStr := string(crypto.StringfyMap(convertValuesToMap(c.Request.URL.Query())))
		// 1. 验证请求是否签名是否正确
		dataToVerify := map[string]string{
			"session":       extendData.SessionId,
			"nonce":         extendData.Nonce,
			"timestamp":     extendData.Timestamp,
			"platform":      fmt.Sprintf("%d", extendData.Platform),
			"method":        method,
			"path":          c.Request.URL.Path,
			"query":         queryStr,
			"authorization": authorization,
		}
		if headers.GetPlatform(c) == core.Web {
			delete(dataToVerify, "authorization")
		}

		if c.Request.Method != http.MethodGet && c.Request.ContentLength > 0 {
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
		verified, err := crypto.VerifySignMap(keys.SignPubKey, dataToVerify, extendData.Signature)
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
		bodyWriter := httpex.NewBodyWriter(c.Writer, respBuf)
		c.Writer = bodyWriter

		c.Next()
		if c.IsAborted() {
			return
		}

		if c.Writer.Status() < 200 || c.Writer.Status() >= 300 {
			return
		}

		// 签名响应体
		responseBody := bodyWriter.GetBytes()
		respTimestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		respNonce := core.NewUUIDWithoutDash()
		dataToSign := map[string]string{
			"session":   extendData.SessionId,
			"nonce":     respNonce,
			"platform":  fmt.Sprintf("%d", extendData.Platform),
			"timestamp": respTimestamp,
			"method":    method,
			"path":      c.Request.URL.Path,
			"query":     queryStr,
			"body":      string(responseBody),
		}

		// 生成响应签名
		respSignature, err := crypto.SignMap(dataToSign)
		if err != nil {
			c.AbortWithError(500, errors.New("sign fail"))
			return
		}

		// 8. 写入响应头
		c.Header(headers.HeaderTimestamp, respTimestamp)
		c.Header(headers.HeaderNonce, respNonce)
		c.Header(headers.HeaderSignature, respSignature)
		// c.Header(headers.HeaderContentLength, strconv.Itoa(len(responseBody)))

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
