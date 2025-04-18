package middleware

import (
	"errors"
	"goapp/internal/app/global"
	"goapp/internal/app/service"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

func JwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		svc := service.NewAuthService()
		// 解析客户端的Token（如果有）
		token := ""
		tokens := svc.GetAuthorizationHeader(c)
		if len(tokens) == 2 && tokens[0] == "Bearer" && len(tokens[1]) > 0 {
			token = tokens[1]
		}

		// web单独处理
		if getPlatform(c) == niu.Web {
			token, _ = c.Cookie(global.AppConfig.Authenticator.Jwt.CookieAccessTokenKey)
		}
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
			claims, err := svc.ParseToken(token)
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
		} else if len(token) > 0 {
			revoked, _ := svc.IsTokenRevoked(c, token)
			if !revoked {
				// 解析Token
				claims, err := svc.ParseToken(token)
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
