package handlers

import (
	"fmt"
	"goapp/internal/app/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

func RegisterAuthHandlers(r *gin.RouterGroup) {
	authGroup := r.Group("/auth", func(c *gin.Context) {

		// TODO: 此处还需要验证该用户的角色
		c.Next()
	})

	authGroup.POST("/negotiate", handleNegotiate)
	authGroup.POST("/login", handleLogin)
	authGroup.POST("/refresh", handleRefresh)
}

// 协商会话密钥
func handleNegotiate(c *gin.Context) {
	// svr := service.NewAuthService()
}

// 手机验证码登录
func handleLogin(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	svr := service.NewAuthService()
	platform := niu.ParsePlatform(svr.GetPlatform(c))
	if !niu.IsPlatformValid(platform) {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("invalid platform"))
		return
	}

	reply := svr.Authorize(c, &req, platform)

	// c.SetSameSite(http.SameSiteLaxMode)
	// c.SetCookie("x-csrf-token", "22222", 0, "/", "", false, false)
	c.JSON(200, reply)
}

// 刷新Token
func handleRefresh(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	svr := service.NewAuthService()
	platform := niu.ParsePlatform(svr.GetPlatform(c))
	if !niu.IsPlatformValid(platform) {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("invalid platform"))
		return
	}

	reply := svr.Authorize(c, &req, platform)

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("x-csrf-token", "22222", 0, "/", "", false, false)
	c.JSON(200, reply)
}
