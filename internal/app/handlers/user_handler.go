package handlers

import "github.com/gin-gonic/gin"

func RegisterUserRoutes(r *gin.Engine) {
	g := r.Group("/usr")
	g.GET("/:id", func(ctx *gin.Context) {
		// Get user info
	})
	g.POST("/:id", func(ctx *gin.Context) {
		// update user info
	})
}
