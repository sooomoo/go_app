package handlers

import "github.com/gin-gonic/gin"

func RegisterRouters(r *gin.Engine) {
	RegisterUserRoutes(r)
}
