package handlers

import (
	"fmt"
	"goapp/internal/app/global"
	"goapp/internal/app/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

// 手机验证码登录
func HandleLogin(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	platform := niu.ParsePlatform(global.GetAuthenticator().GetPlatform(c))
	if !niu.IsPlatformValid(platform) {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("invalid platform"))
		return
	}

	svr := services.NewAuthService()
	reply := svr.Authorize(c, &req, platform)

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("x-csrf-token", "22222", 0, "/", "", false, false)
	c.JSON(200, reply)
}
