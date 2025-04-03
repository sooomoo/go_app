package middleware

import (
	"errors"
	"goapp/internal/app/service"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

func ReplayMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce := strings.TrimSpace(c.GetHeader("X-Nonce"))
		timestampStr := strings.TrimSpace(c.GetHeader("X-Timestamp"))
		platform := strings.TrimSpace(c.GetHeader("X-Platform"))
		signature := strings.TrimSpace(c.GetHeader("X-Signature"))
		sessionId := strings.TrimSpace(c.GetHeader("X-Session"))
		if len(nonce) == 0 || len(timestampStr) == 0 || len(signature) == 0 || len(sessionId) == 0 || !niu.IsPlatformStringValid(platform) {
			c.AbortWithStatus(400)
			return
		}

		// 1. 验证请求是否是重放请求
		svc := service.NewAuthService()
		if svc.IsReplayRequest(c, nonce, timestampStr) {
			c.AbortWithError(400, errors.New("repeat request"))
			return
		}

		c.Next()
	}
}
