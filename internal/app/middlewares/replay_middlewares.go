package middlewares

import (
	"errors"
	"goapp/internal/app/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 防止重放攻击的中间件
func ReplayMiddleware(c *gin.Context) {
	authSvr := services.NewAuthService()
	if authSvr.IsReplayRequest(c) {
		c.AbortWithError(http.StatusForbidden, errors.New("no replay"))
		return
	}
	c.Next()
}
