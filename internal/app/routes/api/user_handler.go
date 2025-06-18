package api

import (
	"goapp/internal/app/services"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
}

var (
	userHandler *UserHandler = &UserHandler{}
)

func (u *UserHandler) RegisterRoutes(router *gin.RouterGroup) {
	g := router.Group("/user")
	g.GET("/info", u.handleGetSelfUserInfo)
	g.POST("/:id", func(ctx *gin.Context) {
		// update user info
	})
}

func (u *UserHandler) handleGetSelfUserInfo(c *gin.Context) {
	user, err := services.NewUserService().GetSelfInfo(c)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.JSON(200, user)
}
