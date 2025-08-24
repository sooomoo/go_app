package authes

import (
	"fmt"
	"goapp/internal/app/features/users"
	"goapp/internal/app/global"
	"goapp/internal/app/models"
	"goapp/internal/app/shared"
	"goapp/internal/app/shared/headers"
	"goapp/pkg/core"
	"goapp/pkg/strs"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MsgCodeAuthService struct {
	*AuthService
}

func NewMsgCodeAuthService() *MsgCodeAuthService {
	return &MsgCodeAuthService{
		AuthService: &AuthService{
			authRepo: NewAuthStore(),
			userRepo: users.NewUserStore(),
		},
	}
}

// 生成随机验证码
func (a *MsgCodeAuthService) GenerateMsgCode() string {
	code := rand.Intn(9000) + 1000 // 生成4位数验证码
	return fmt.Sprintf("%04d", code)
}

type MsgCodeLoginRequest struct {
	CountryCode string `json:"countryCode" binding:"required"`
	Phone       string `json:"phone" binding:"required"`
	ImgCode     string `json:"imgCode" binding:"required"`
	MsgCode     string `json:"msgCode" binding:"required"`
	CsrfToken   string `json:"csrfToken" binding:"required"`
}

func (a *MsgCodeAuthService) Authorize(ctx *gin.Context, req *MsgCodeLoginRequest) *AuthResponseDto {
	csrfToken := headers.GetCsrfToken(ctx)
	if csrfToken != req.CsrfToken {
		ctx.AbortWithStatus(400)
		return nil
	}
	if !strs.IsCountryCode(req.CountryCode) || !strs.IsCellPhone(req.Phone) {
		ctx.AbortWithStatus(400)
		return nil
	}
	// 从缓存中取图形验证码
	imgCode, err := a.authRepo.GetCsrfToken(ctx, csrfToken, true)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}
	// 验证验证码
	if req.ImgCode != imgCode {
		ctx.AbortWithStatus(400)
		return nil
	}
	// 验证安全码
	if req.MsgCode != "8888" {
		ctx.AbortWithStatus(400)
		return nil
	}

	// 通过手机号注册或获取用户信息
	ip := ctx.ClientIP()
	fullphone := req.CountryCode + req.Phone
	user, err := a.userRepo.Upsert(ctx, fullphone, ip)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}
	// 该用户已被禁用
	if user.Status == models.UserStatusBanned {
		ctx.AbortWithStatus(403)
		return nil
	}

	// 生成token, 将这些Token与该用户绑定
	accessToken, refreshToken, err := a.GenerateTokenPair(ctx, user.ID)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}

	jwtConfig := global.GetAuthConfig().Jwt
	ctx.SetSameSite(http.SameSite(jwtConfig.CookieSameSiteMode))
	ctx.SetCookie(headers.CookieKeyCsrfToken, "", -1000, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, jwtConfig.CookieHttpOnly)

	if headers.GetPlatform(ctx) == core.Web {
		clientId := headers.GetClientId(ctx)
		a.setupAuthorizedCookie(ctx, clientId, accessToken, refreshToken)
		return &AuthResponseDto{Code: shared.RespCodeSucceed}
	}

	return &AuthResponseDto{Code: shared.RespCodeSucceed, Data: &AuthResponse{accessToken, refreshToken}}
}
