package service

import (
	"errors"
	"goapp/internal/app/global"
	"goapp/internal/app/repository"
	"goapp/internal/app/repository/dao/model"
	"goapp/internal/app/service/headers"

	"github.com/gin-gonic/gin"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService() *UserService {
	return &UserService{
		userRepo: repository.NewUserRepository(global.Cache, global.Db),
	}
}

func (u *UserService) GetSelfInfo(c *gin.Context) (*model.User, error) {
	claims := headers.GetClaims(c)
	if claims == nil {
		return nil, errors.New("not found")
	}
	return u.userRepo.GetById(c, int64(claims.UserId))
}
