package services

import (
	"context"
	"goapp/internal/app/global"
	"goapp/internal/app/repositories"

	"github.com/sooomo/niu"
)

type LoginRequest struct {
	Phone      string `json:"phone" binding:"required"`
	Code       string `json:"code" binding:"required"`
	SecureCode string `json:"secure_code" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type AuthService struct {
}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func (s *AuthService) Authorize(ctx context.Context, req *LoginRequest, platform niu.Platform) *niu.ReplyDto[ReplyCode, LoginResponse] {
	reply := &niu.ReplyDto[ReplyCode, LoginResponse]{}
	// 验证验证码
	if req.Code != "1234" {
		reply.Code = ReplyCodeInvalidMsgCode
		return reply
	}
	// 验证安全码
	if req.SecureCode != "8888" {
		reply.Code = ReplyCodeInvalidSecureCode
		return reply
	}

	// 通过手机号注册或获取用户信息
	repo := repositories.NewRepositoryOfUser(nil, nil)
	user, err := repo.Upsert(req.Phone)
	if err != nil {
		reply.Code = ReplyCodeFailed
		reply.Msg = err.Error()
		return reply
	}

	// 生成token
	token, err := global.GetAuthenticator().GenerateToken(user.Id, user.Roles, platform)
	if err != nil {
		reply.Code = ReplyCodeFailed
		reply.Msg = err.Error()
		return reply
	}

	reply.Code = ReplyCodeSucceed
	reply.Data = LoginResponse{Token: token}
	return reply
}
