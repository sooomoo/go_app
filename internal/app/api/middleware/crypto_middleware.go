package middleware

// import (
// 	"bytes"
// 	"goapp/internal/app/config"
// 	"goapp/internal/app/util"
// 	"io"
// 	"net/http"
// 	"strconv"
// 	"strings"

// 	"github.com/gin-gonic/gin"
// 	"github.com/sooomo/niu"
// )

// // 请求体结构
// type SignedRequest struct {
// 	Data      string `json:"data"`      // 加密后的数据
// 	Signature string `json:"signature"` // 签名
// }

// // 响应体结构
// type SignedResponse struct {
// 	Data      string `json:"data"`      // 加密后的数据
// 	Signature string `json:"signature"` // 签名
// }

// // 请求解密和验签中间件
// func RequestDecryptor() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		// 跳过GET请求
// 		if c.Request.Method == http.MethodGet {
// 			c.Next()
// 			return
// 		}

// 		// 读取请求体
// 		bodyBytes, err := io.ReadAll(c.Request.Body)
// 		if err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无法读取请求体"})
// 			c.Abort()
// 			return
// 		}

// 		// 解密数据
// 		decryptedData, err := util.Decrypt(string(bodyBytes), config.AppConfig.Crypto.EncryptKey)
// 		if err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "数据解密失败"})
// 			c.Abort()
// 			return
// 		}

// 		// 替换请求体
// 		c.Request.Body = io.NopCloser(bytes.NewBufferString(decryptedData))
// 		c.Next()
// 	}
// }

// // 响应加密和签名中间件
// func ResponseEncryptor() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		// 创建自定义ResponseWriter
// 		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
// 		c.Writer = blw

// 		// 处理请求
// 		c.Next()

// 		// 跳过非JSON响应
// 		contentType := c.Writer.Header().Get("Content-Type")
// 		if !strings.Contains(contentType, "application/json") {
// 			return
// 		}

// 		// 获取原始响应
// 		responseBody := blw.body.String()

// 		// 加密响应数据
// 		encryptedData, err := util.Encrypt(responseBody, config.AppConfig.Crypto.EncryptKey)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "响应加密失败"})
// 			return
// 		}

// 		// 重置响应
// 		c.Header("Content-Type", niu.ContentTypeEncrypted)
// 		c.Header("Content-Length", strconv.Itoa(len(responseBody)))
// 		c.Status(http.StatusOK)
// 		blw.ResponseWriter.WriteString(encryptedData)
// 	}
// }

// // 自定义ResponseWriter，用于捕获响应体
// type bodyLogWriter struct {
// 	gin.ResponseWriter
// 	body *bytes.Buffer
// }

// func (w bodyLogWriter) Write(b []byte) (int, error) {
// 	return w.body.Write(b)
// }
