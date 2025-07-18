package middleware

import (
	"errors"
	"goapp/internal/app"
	"goapp/internal/app/services"
	"goapp/internal/app/services/headers"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		svc := services.NewAuthService()
		// 解析客户端的Token（如果有）
		token := headers.GetAccessToken(c)
		// 解析Token
		claims, err := svc.ParseAccessToken(c, token)
		claimsValid := svc.IsClaimsValid(c, claims)
		if claimsValid {
			headers.SaveClaims(c, claims)
		}
		if isPathNeedAuth(c.Request.URL.Path) && !claimsValid {
			if err != nil && err != redis.Nil {
				c.AbortWithError(500, errors.New("parse token fail"))
				return
			}
			c.AbortWithError(401, errors.New("invalid token"))
			return
		}

		// 解析并存储客户端的Key
		headers.SaveClientKeys(c)
		if c.IsAborted() {
			return
		}

		c.Next()
	}
}

func isPathNeedAuth(path string) bool {
	for _, p := range app.GetGlobal().GetAuthConfig().PathsNotAuth {
		if strings.EqualFold(p, path) {
			return false
		}
	}
	for _, p := range app.GetGlobal().GetAuthConfig().PathsNeedAuth {
		if strings.Contains(p, "*") {
			return true
		}
		if strings.EqualFold(p, path) {
			return true
		}
	}
	return false
}
