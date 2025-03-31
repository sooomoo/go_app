package handlers

import (
	"goapp/internal/app/handlers/hubs"

	"github.com/gin-gonic/gin"
)

func RegisterHandlers(r *gin.RouterGroup) {
	RegisterAuthHandlers(r)
	RegisterUserRoutes(r)
	RegisterAdminRoutes(r)

	hubs.RegisterHubs(r)
}
