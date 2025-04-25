package api

import (
	"goapp/internal/app/service"

	"github.com/gin-gonic/gin"
)

func RegisterAuthHandlers(r *gin.RouterGroup) {
	authGroup := r.Group("/auth", func(c *gin.Context) {
		// TODO: 此处还需要验证该用户的角色
		c.Next()
	})

	authGroup.POST("/login/prepare", handleLoginPrepare)
	authGroup.POST("/login/do", handleLoginDo)
	authGroup.POST("/refresh", handleRefresh)
	authGroup.POST("/logout", handleLogout)
}

func handleLoginPrepare(c *gin.Context) {
	svr := service.NewAuthService()
	reply := svr.PrepareLogin(c)
	if c.IsAborted() {
		return
	}

	c.JSON(200, reply)
}

// 手机验证码登录
func handleLoginDo(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, service.NewResponseDtoNoData(service.RespCodeInvalidArgs, ""))
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
	svr := service.NewAuthService()
	svr.Logout(c)
	c.Status(200)
}
