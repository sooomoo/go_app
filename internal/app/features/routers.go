package features

import (
	"goapp/internal/app/features/authes"
	"goapp/internal/app/features/users"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	authes.GetAuthHandler().RegisterRoutes(r)
	users.GetUserHandler().RegisterRoutes(r)
}
