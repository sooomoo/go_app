package services

import (
	"errors"
	"goapp/internal/app"
	"goapp/internal/app/repositories"
	"goapp/internal/app/services/headers"

	"github.com/gin-gonic/gin"
)

type UserService struct {
	userRepo *repositories.UserRepository
}

func NewUserService() *UserService {
	return &UserService{
		userRepo: repositories.NewUserRepository(app.GetGlobal().GetCache()),
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
