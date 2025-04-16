package handlers

import (
	"fmt"
	"goapp/internal/app/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

func RegisterUserRoutes(r *gin.RouterGroup) {
	g := r.Group("/user")
	g.GET("/info", handleGetSelfUserInfo)
	g.POST("/:id", func(ctx *gin.Context) {
		// update user info
	})
}

func handleGetSelfUserInfo(c *gin.Context) {
	user, err := service.NewUserService().GetSelfInfo(c)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	ccc, err := c.Request.Cookie("httponlycc")
	fmt.Print(ccc)

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("httponlycc", niu.NewUUIDWithoutDash(), 7200, "/", "", false, true)

	c.JSON(200, user)
}
