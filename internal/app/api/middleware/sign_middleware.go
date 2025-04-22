package middleware

import (
	"errors"
	"fmt"
	"goapp/internal/app/service"
	"goapp/internal/app/service/headers"
	"goapp/internal/pkg/crypto"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

func SignMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := strings.TrimSpace(c.GetHeader(headers.HeaderAuthorization))

		var extendData *service.RequestExtendData
		extData, ok := c.Get(service.KeyExtendData)
		if ok {
			extendData = extData.(*service.RequestExtendData)
		}

		keys := getClientKeys(c)
		if keys == nil {
			c.AbortWithError(400, errors.New("bad skeys"))
			return
		}

		// 1. 验证请求是否签名是否正确
		dataToVerify := map[string]string{
			"session":       extendData.SessionId,
			"nonce":         extendData.Nonce,
			"timestamp":     extendData.Timestamp,
			"platform":      fmt.Sprintf("%d", extendData.Platform),
			"method":        c.Request.Method,
			"path":          c.Request.URL.Path,
			"query":         string(crypto.StringfyMap(convertValuesToMap(c.Request.URL.Query()))),
			"authorization": authorization,
		}
		if headers.GetPlatform(c) == niu.Web {
			delete(dataToVerify, "authorization")
		}

		if c.Request.Method != http.MethodGet {
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
			"session":   extendData.SessionId,
			"nonce":     respNonce,
			"platform":  fmt.Sprintf("%d", extendData.Platform),
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
		c.Header(headers.HeaderTimestamp, respTimestamp)
		c.Header(headers.HeaderNonce, respNonce)
		c.Header(headers.HeaderSignature, respSignature)
		c.Header(headers.HeaderContentLength, strconv.Itoa(len(responseBody)))

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
