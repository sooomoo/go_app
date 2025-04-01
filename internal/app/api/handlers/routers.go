package handlers

import (
	"goapp/internal/app/api/handlers/hubs"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	RegisterAuthHandlers(r)
	RegisterUserRoutes(r)
	RegisterAdminRoutes(r)

	hubs.RegisterHubs(r)

}
