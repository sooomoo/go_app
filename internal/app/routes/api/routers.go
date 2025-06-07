package api

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	authHandler.RegisterRoutes(r)
	adminHandler.RegisterRoutes(r)
	userHandler.RegisterRoutes(r)
}
