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
		tokens := svc.GetAuthorizationHeader(c)
		isTokenValid := len(tokens) == 2 && tokens[0] == "Bearer" && len(tokens[1]) > 0
		if isPathNeedAuth(c.Request.URL.Path) {
			if !isTokenValid {
				c.AbortWithStatus(401)
				return
			}

			revoked, err := svc.IsTokenRevoked(c, tokens[1])
			if err != nil {
				c.AbortWithError(500, errors.New("check token revoke fail"))
				return
			}
			if revoked {
				c.AbortWithStatus(401)
				return
			}
			// 解析Token
			claims, err := svc.ParseToken(tokens[1])
			if err != nil || claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
				c.AbortWithError(401, errors.New("invalid token"))
				return
			}

			// 刷新Token时，此处的类型为 r
			allowTokenTyep := "a"
			if strings.EqualFold(c.Request.URL.Path, global.AppConfig.Authenticator.RefreshTokenPath) {
				allowTokenTyep = "r"
			}
			if claims.Type != allowTokenTyep {
				c.AbortWithError(401, errors.New("invalid token type"))
				return
			}

			svc.SaveClaims(c, claims)
		} else if isTokenValid {
			revoked, _ := svc.IsTokenRevoked(c, tokens[1])
			if !revoked {
				// 解析Token
				claims, err := svc.ParseToken(tokens[1])
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
