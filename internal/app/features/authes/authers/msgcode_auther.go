package authers

import (
	"fmt"
	"goapp/internal/app/features/authes/stores"
	"goapp/internal/app/features/users"
	"goapp/internal/app/models"
	"goapp/internal/app/shared/headers"
	"goapp/pkg/strs"
	"math/rand"

	"github.com/gin-gonic/gin"
)

type MsgCodeAuther struct {
	authRepo *stores.AuthStore
	userRepo *users.UserStore
}

func NewMsgCodeAuther() *MsgCodeAuther {
	return &MsgCodeAuther{
		authRepo: stores.NewAuthStore(),
		userRepo: users.NewUserStore(),
	}
}

type MsgCodeLoginRequest struct {
	CountryCode string `json:"countryCode" binding:"required"`
	Phone       string `json:"phone" binding:"required"`
	ImgCode     string `json:"imgCode" binding:"required"`
	MsgCode     string `json:"msgCode" binding:"required"`
	CsrfToken   string `json:"csrfToken" binding:"required"`
}

// 生成随机验证码
func (a *MsgCodeAuther) GenerateMsgCode() string {
	code := rand.Intn(9000) + 1000 // 生成4位数验证码
	return fmt.Sprintf("%04d", code)
}

func (a *MsgCodeAuther) Authorize(ctx *gin.Context, r AuthRequest) *models.User {
	req, ok := r.(*MsgCodeLoginRequest)
	if !ok {
		ctx.AbortWithStatus(400)
		return nil
	}
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
	return user
}
