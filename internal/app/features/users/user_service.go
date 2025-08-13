package users

import (
	"errors"
	"goapp/internal/app/shared"
	"goapp/internal/app/shared/claims"
	"goapp/pkg/ids"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo *UserStore
}

func NewUserService() *UserService {
	return &UserService{
		userRepo: NewUserStore(),
	}
}

type GetUserInfoResponse struct {
	Id        ids.UID `json:"id"`
	Name      string  `json:"name"`
	AvatarUrl string  `json:"avatarUrl"`
	Role      int32   `json:"role"`
	IpLatest  string  `json:"ipLatest"`
}
type GetUserInfoResponseDto shared.ResponseDto[*GetUserInfoResponse]

func (u *UserService) GetSelfInfo(c *gin.Context) (*GetUserInfoResponseDto, error) {
	cc := claims.GetClaims(c)
	if cc == nil {
		return nil, errors.New("not found")
	}
	user, err := u.userRepo.GetById(c, cc.UserId)
	if err == gorm.ErrRecordNotFound {
		c.AbortWithStatus(401)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// convert
	return &GetUserInfoResponseDto{
		Code: shared.RespCodeSucceed,
		Msg:  "succeed",
		Data: &GetUserInfoResponse{
			Id:        user.ID,
			Name:      user.Name,
			AvatarUrl: user.Profiles.GetString("avatar", ""),
			Role:      user.Role,
			IpLatest:  user.IP.GetString("latest", ""),
		},
	}, nil
}
