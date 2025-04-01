package middleware

import (
	"net/http"
	"strings"

	"goapp/internal/app/service"

	"github.com/gin-gonic/gin"
)

// JWT认证中间件
func JWTAuth() gin.HandlerFunc {
	authService := service.NewAuthService()
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证令牌"})
			c.Abort()
			return
		}

		// 检查Bearer前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "认证令牌格式错误"})
			c.Abort()
			return
		}

		// 解析令牌
		claims, err := authService.ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的认证令牌"})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		authService.SaveClaims(c, claims)
		c.Next()
	}
}
