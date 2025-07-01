package services

import (
	"errors"
	"goapp/internal/app"
	"goapp/internal/app/services/headers"
	"goapp/internal/app/stores"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo *stores.UserStore
}

func NewUserService() *UserService {
	return &UserService{
		userRepo: stores.NewUserStore(app.GetGlobal().GetCache()),
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
	user, err := u.userRepo.GetById(c, claims.UserId)
	if err == gorm.ErrRecordNotFound {
		c.AbortWithStatus(401)
		return nil, nil
	}
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
			AvatarUrl: user.Profiles,
			Role:      user.Role,
			IpLatest:  user.IP,
		},
	}, nil
}
