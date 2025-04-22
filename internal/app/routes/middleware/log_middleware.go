package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// LogMiddleware 请求日志中间件
func LogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()

		// 执行时间
		latency := endTime.Sub(startTime)

		// 请求方法
		reqMethod := c.Request.Method

		// 请求路由
		reqURI := c.Request.RequestURI

		// 状态码
		statusCode := c.Writer.Status()

		// 请求IP
		clientIP := c.ClientIP()

		// 日志格式
		fmt.Printf("[API] %v | %3d | %8vms | %15s | %s | %s\n",
			endTime.Format("2006/01/02 15:04:05"),
			statusCode,
			latency.Milliseconds(),
			clientIP,
			reqMethod,
			reqURI,
		)
	}
}
