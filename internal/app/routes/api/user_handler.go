package api

import (
	"goapp/internal/app/service"

	"github.com/gin-gonic/gin"
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

	c.JSON(200, user)
}
