package handlers

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	RegisterAuthHandlers(r)
	RegisterUserRoutes(r)
	RegisterAdminRoutes(r)
}
