package handlers

import (
	"goapp/internal/app/service"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

func RegisterAuthHandlers(r *gin.RouterGroup) {
	authGroup := r.Group("/auth", func(c *gin.Context) {
		// TODO: 此处还需要验证该用户的角色
		c.Next()
	})

	authGroup.POST("/login", handleLogin)
	authGroup.POST("/refresh", handleRefresh)
	authGroup.POST("/logout", handleLogout)
}

// 手机验证码登录
func handleLogin(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, &niu.ReplyDto[service.RespCode, any]{
			Code: service.RespCodeInvalidArgs,
		})
		return
	}

	svr := service.NewAuthService()
	reply := svr.Authorize(c, &req)
	if c.IsAborted() {
		return
	}

	c.JSON(200, reply)
}

// 刷新Token
func handleRefresh(c *gin.Context) {
	svr := service.NewAuthService()
	reply := svr.RefreshToken(c)
	if c.IsAborted() {
		return
	}
	c.JSON(200, reply)
}

// 退出登录
func handleLogout(c *gin.Context) {
	c.Status(200)
}
