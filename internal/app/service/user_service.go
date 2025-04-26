package service

import (
	"errors"
	"goapp/internal/app/global"
	"goapp/internal/app/repository"
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

type GetUserInfoResponse struct {
	Id        int64  `json:"id"`
	Name      string `json:"name"`
	AvatarUrl string `json:"avatarUrl"`
	Role      int32  `json:"role"`
	IpLatest  string `json:"ipLatest"`
}
type GetUserInfoResponseDto ResponseDto[*GetUserInfoResponse]

func (u *UserService) GetSelfInfo(c *gin.Context) (*GetUserInfoResponseDto, error) {
	claims := headers.GetClaims(c)
	if claims == nil {
		return nil, errors.New("not found")
	}
	user, err := u.userRepo.GetById(c, int64(claims.UserId))
	if err != nil {
		return nil, err
	}
	// convert
	return &GetUserInfoResponseDto{
		Code: RespCodeSucceed,
		Msg:  "succeed",
		Data: &GetUserInfoResponse{
			Id:        user.ID,
			Name:      user.Name,
			AvatarUrl: user.AvatarURL,
			Role:      user.Role,
			IpLatest:  user.IPLatest,
		},
	}, nil
}
