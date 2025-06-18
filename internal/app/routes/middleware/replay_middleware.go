package middleware

import (
	"errors"
	"goapp/internal/app/services"
	"goapp/internal/app/services/headers"
	"goapp/pkg/core"

	"github.com/gin-gonic/gin"
)

func ReplayMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		svc := services.NewAuthService()
		nonce := headers.GetTrimmedHeader(c, headers.HeaderNonce)
		timestampStr := headers.GetTrimmedHeader(c, headers.HeaderTimestamp)
		platform := headers.GetPlatform(c)
		signature := headers.GetTrimmedHeader(c, headers.HeaderSignature)
		sessionId := headers.GetSessionId(c)
		if len(nonce) == 0 || len(timestampStr) == 0 || len(signature) == 0 || len(sessionId) == 0 || !core.IsPlatformValid(platform) {
			c.AbortWithStatus(400)
			return
		}

		// 1. 验证请求是否是重放请求
		if svc.IsReplayRequest(c, nonce, timestampStr) {
			c.AbortWithError(400, errors.New("repeat request"))
			return
		}

		headers.SaveExtendData(c)

		c.Next()
	}
}
