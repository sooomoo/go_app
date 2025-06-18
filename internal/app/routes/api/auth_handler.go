package api

import (
	"goapp/internal/app/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
}

var (
	authHandler *AuthHandler = &AuthHandler{}
)

func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	authGroup := router.Group("/auth", func(c *gin.Context) {
		// TODO: 此处还需要验证该用户的角色
		c.Next()
	})

	authGroup.POST("/login/prepare", h.handleLoginPrepare)
	authGroup.POST("/login/do", h.handleLoginDo)
	authGroup.POST("/refresh", h.handleRefresh)
	authGroup.POST("/logout", h.handleLogout)
}

func (h *AuthHandler) handleLoginPrepare(c *gin.Context) {
	svr := services.NewAuthService()
	reply := svr.PrepareLogin(c)
	if c.IsAborted() {
		return
	}

	c.JSON(200, reply)
}

// 手机验证码登录
func (h *AuthHandler) handleLoginDo(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, services.NewResponseInvalidArgs(""))
		return
	}

	svr := services.NewAuthService()
	reply := svr.Authorize(c, &req)
	if c.IsAborted() {
		return
	}

	c.JSON(200, reply)
}

// 刷新Token
func (h *AuthHandler) handleRefresh(c *gin.Context) {
	svr := services.NewAuthService()
	reply := svr.RefreshToken(c)
	if c.IsAborted() {
		return
	}
	c.JSON(200, reply)
}

// 退出登录
func (h *AuthHandler) handleLogout(c *gin.Context) {
	svr := services.NewAuthService()
	svr.Logout(c)
	c.Status(200)
}
