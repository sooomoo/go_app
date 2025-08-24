package authers

import (
	"goapp/internal/app/models"

	"github.com/gin-gonic/gin"
)

type AuthRequest any

type Auther interface {
	Authorize(ctx *gin.Context, req AuthRequest) *models.User
}
