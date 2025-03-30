package handlers

import (
	"goapp/internal/app/handlers/hubs"

	"github.com/gin-gonic/gin"
)

func RegisterHandlers(r *gin.RouterGroup) {
	RegisterUserRoutes(r)
	RegisterAdminRoutes(r)

	hubs.RegisterHubs(r)
}
