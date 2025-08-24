package authers

import (
	"goapp/internal/app/models"

	"github.com/gin-gonic/gin"
)

type PasswordAuther struct {
}

func NewPasswordAuther() *PasswordAuther {
	return &PasswordAuther{}
}

type PasswordLoginRequest struct {
	CountryCode string `json:"countryCode" binding:"required"`
	Phone       string `json:"phone" binding:"required"`
	ImgCode     string `json:"imgCode" binding:"required"`
	Password    string `json:"password" binding:"required"`
	CsrfToken   string `json:"csrfToken" binding:"required"`
}

func (a *PasswordAuther) Authorize(ctx *gin.Context, r AuthRequest) *models.User {
	return nil
}
