package middleware

import (
	"errors"
	"goapp/internal/app/global"
	"goapp/internal/app/service"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func JwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		svc := service.NewAuthService()
		// 解析客户端的Token（如果有）
		token := svc.GetAccessToken(c)
		if isPathNeedAuth(c.Request.URL.Path) {
			if len(token) == 0 {
				c.AbortWithStatus(401)
				return
			}

			revoked, err := svc.IsTokenRevoked(c, token)
			if err != nil {
				c.AbortWithError(500, errors.New("check token revoke fail"))
				return
			}
			if revoked {
				c.AbortWithStatus(401)
				return
			}
			// 解析Token
			claims, err := svc.ParseAccessToken(token)
			ua := svc.GetUserAgentHashed(c)
			if err != nil || claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) || claims.UserAgent != ua {
				c.AbortWithError(401, errors.New("invalid token"))
				return
			}

			svc.SaveClaims(c, claims)
		} else if len(token) > 0 {
			revoked, _ := svc.IsTokenRevoked(c, token)
			if !revoked {
				// 解析Token
				claims, err := svc.ParseAccessToken(token)
				if err != nil || claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
					// 忽略错误
					svc.SaveClaims(c, claims)
				}
			}
		}

		// 解析并存储客户端的Key
		parseAndStoreClientKeys(c, strings.TrimSpace(c.GetHeader("X-Session")))
		if c.IsAborted() {
			return
		}

		c.Next()
	}
}

func isPathNeedAuth(path string) bool {
	for _, p := range global.AppConfig.Authenticator.PathsNotAuth {
		if strings.EqualFold(p, path) {
			return false
		}
	}
	for _, p := range global.AppConfig.Authenticator.PathsNeedAuth {
		if strings.Contains(p, "*") {
			return true
		}
		if strings.EqualFold(p, path) {
			return true
		}
	}
	return false
}
