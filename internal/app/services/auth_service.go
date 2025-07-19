package services

import (
	"context"
	"errors"
	"fmt"
	"goapp/internal/app"
	"goapp/internal/app/services/headers"
	"goapp/internal/app/stores"
	"goapp/pkg/core"
	"goapp/pkg/strs"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
)

type AuthService struct {
	authRepo *stores.AuthStore
	userRepo *stores.UserStore
}

func NewAuthService() *AuthService {
	return &AuthService{
		authRepo: stores.NewAuthStore(),
		userRepo: stores.NewUserStore(),
	}
}

type PrepareLoginResponse struct {
	CsrfToken string `json:"csrfToken"`
	ImageData string `json:"imageData"`
}

type PrepareLoginResponseDto = ResponseDto[*PrepareLoginResponse]

var captchaDriver = base64Captcha.NewDriverDigit(40, 80, 4, 0.5, 60)

func (a *AuthService) PrepareLogin(ctx *gin.Context) *PrepareLoginResponseDto {
	// 生成随机图形验证码
	id, content, answer := captchaDriver.GenerateIdQuestionAnswer()
	fmt.Printf("captchaId: %s, q: %s, answer: %s\n", id, content, answer)
	// 生成 Base64 编码的验证码图片
	item, err := captchaDriver.DrawCaptcha(content)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}
	base64Str := item.EncodeB64string()
	// 生成csrf token
	csrfToken := core.NewUUID()
	// 将验证码存入缓存中
	dur := 10 * time.Minute
	err = a.authRepo.SaveCsrfToken(ctx, csrfToken, answer, dur)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}
	jwtConfig := app.GetGlobal().GetAuthConfig().Jwt
	ctx.SetSameSite(http.SameSite(jwtConfig.CookieSameSiteMode))
	ctx.SetCookie(headers.CookieKeyCsrfToken, csrfToken, int(dur.Seconds()), "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, jwtConfig.CookieHttpOnly)
	// 返回验证码和csrf token
	return &PrepareLoginResponseDto{Code: RespCodeSucceed, Data: &PrepareLoginResponse{CsrfToken: csrfToken, ImageData: base64Str}}
}

type LoginRequest struct {
	CountryCode string `json:"countryCode" binding:"required"`
	Phone       string `json:"phone" binding:"required"`
	ImgCode     string `json:"imgCode" binding:"required"`
	MsgCode     string `json:"msgCode" binding:"required"`
	CsrfToken   string `json:"csrfToken" binding:"required"`
}

// AuthResponse JWT令牌对
type AuthResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type AuthResponseDto = ResponseDto[*AuthResponse]

func (a *AuthService) Authorize(ctx *gin.Context, req *LoginRequest) *AuthResponseDto {
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
	if user.Status == stores.UserStatusBlock {
		ctx.AbortWithStatus(403)
		return nil
	}

	// 生成token, 将这些Token与该用户绑定
	accessToken, refreshToken, err := a.GenerateTokenPair(ctx, user.ID)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}

	jwtConfig := app.GetGlobal().GetAuthConfig().Jwt
	ctx.SetSameSite(http.SameSite(jwtConfig.CookieSameSiteMode))
	ctx.SetCookie(headers.CookieKeyCsrfToken, "", -1000, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, jwtConfig.CookieHttpOnly)

	if headers.GetPlatform(ctx) == core.Web {
		clientId := headers.GetClientId(ctx)
		a.setupAuthorizedCookie(ctx, clientId, accessToken, refreshToken)
		return &AuthResponseDto{Code: RespCodeSucceed}
	}

	return &AuthResponseDto{Code: RespCodeSucceed, Data: &AuthResponse{accessToken, refreshToken}}
}

type RefreshTokenRequest struct {
	Token string `json:"token"`
}

func (a *AuthService) RefreshToken(ctx *gin.Context) *AuthResponseDto {
	token := headers.GetRefreshToken(ctx)
	if len(token) == 0 {
		var req RefreshTokenRequest
		reqErr := ctx.ShouldBindJSON(&req)
		if reqErr == nil {
			token = req.Token
		}
	}

	if len(token) == 0 {
		ctx.AbortWithStatus(401)
		return nil
	}

	// accessToken如果存在，就吊销
	accessToken := headers.GetAccessToken(ctx)
	a.RevokeAccessToken(ctx, accessToken)

	credentials := a.authRepo.GetRefreshTokenCredential(ctx, token)
	if credentials == nil {
		ctx.AbortWithStatus(401) // client need re-login
		return nil
	}

	if !a.IsClaimsValid(ctx, credentials) {
		ctx.AbortWithStatus(401) // client need re-login
		return nil
	}

	err := a.RevokeRefreshToken(ctx, token)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}

	// 轮换 clientid 与 refresh token
	accessToken, refreshToken, err := a.GenerateTokenPair(ctx, credentials.UserId)
	if err != nil {
		ctx.AbortWithStatus(401) // token 已经删除，此时只能重新登录
		return nil
	}

	if headers.GetPlatform(ctx) == core.Web {
		clientId := headers.GetClientId(ctx)
		a.setupAuthorizedCookie(ctx, clientId, accessToken, refreshToken)
		return &AuthResponseDto{Code: RespCodeSucceed}
	}

	return &AuthResponseDto{Code: RespCodeSucceed, Data: &AuthResponse{accessToken, refreshToken}}
}

func (a *AuthService) Logout(ctx *gin.Context) {
	accessToken := headers.GetAccessToken(ctx)
	refreshToken := headers.GetRefreshToken(ctx)

	a.RevokeAccessToken(ctx, accessToken)
	a.RevokeRefreshToken(ctx, refreshToken)

	// 关闭所有的 hub
	// TODO: 可能不需要，客户端主动关闭也行

	// 删除 cookie
	jwtConfig := app.GetGlobal().GetAuthConfig().Jwt
	ctx.SetSameSite(http.SameSite(jwtConfig.CookieSameSiteMode))
	ctx.SetCookie(headers.CookieKeyAccessToken, "", -1000, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, jwtConfig.CookieHttpOnly)
	ctx.SetCookie(headers.CookieKeyRefreshToken, "", -1000, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, jwtConfig.CookieHttpOnly)
}

func (a *AuthService) setupAuthorizedCookie(ctx *gin.Context, clientId, accessToken, refreshToken string) {
	jwtConfig := app.GetGlobal().GetAuthConfig().Jwt
	ctx.SetSameSite(http.SameSite(jwtConfig.CookieSameSiteMode))
	atkMaxAge := int((time.Duration(jwtConfig.AccessTtl) * time.Minute).Seconds())
	rtkMaxAge := int((time.Duration(jwtConfig.RefreshTtl) * time.Minute).Seconds())
	cliMaxAge := rtkMaxAge + int((time.Hour * 24 * 30).Seconds())
	ctx.SetCookie(headers.CookieKeyAccessToken, accessToken, atkMaxAge, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, jwtConfig.CookieHttpOnly)
	ctx.SetCookie(headers.CookieKeyRefreshToken, refreshToken, rtkMaxAge, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, jwtConfig.CookieHttpOnly)
	ctx.SetCookie(headers.CookieKeyClientId, clientId, cliMaxAge, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, jwtConfig.CookieHttpOnly)
}

func (a *AuthService) IsClaimsValid(ctx *gin.Context, claims *stores.AuthorizedClaims) bool {
	if claims == nil {
		return false
	}
	clientId := headers.GetClientId(ctx)
	platform := headers.GetPlatform(ctx)
	ua := headers.GetUserAgentHashed(ctx)
	return clientId == claims.ClientId && platform == claims.Platform && ua == claims.UserAgentHashed
}

func (a *AuthService) RevokeAccessToken(ctx context.Context, token string) error {
	if len(token) == 0 {
		return nil
	}
	return a.authRepo.DeleteAccessToken(ctx, token) // 调用Repository层的方法
}

func (a *AuthService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	if len(refreshToken) == 0 {
		return nil
	}
	return a.authRepo.DeleteRefreshToken(ctx, refreshToken) // 调用Repository层的方法
}

func (a *AuthService) GenerateTokenPair(ctx *gin.Context, userID int64) (string, string, error) {
	clientId := headers.GetClientId(ctx)
	platform := headers.GetPlatform(ctx)
	accessToken, claims, err := a.GenerateAccessToken(ctx, userID, clientId, platform)
	if err != nil {
		return "", "", err
	}
	refreshToken := core.NewUUID()
	err = a.authRepo.SaveRefreshToken(ctx, refreshToken, claims, time.Duration(app.GetGlobal().GetAuthConfig().Jwt.RefreshTtl)*time.Minute)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func (a *AuthService) GenerateAccessToken(ctx *gin.Context, userID int64, clientId string, platform core.Platform) (string, *stores.AuthorizedClaims, error) {
	if len(clientId) == 0 || platform == core.Unspecify {
		return "", nil, errors.New("invalid args")
	}
	token := core.NewUUID()
	claims := stores.AuthorizedClaims{
		UserId:          userID,
		Platform:        platform,
		UserAgent:       headers.GetUserAgent(ctx),
		ClientId:        clientId,
		UserAgentHashed: headers.GetUserAgentHashed(ctx),
		Ip:              ctx.ClientIP(),
	}
	err := a.authRepo.SaveAccessToken(ctx, token, time.Duration(app.GetGlobal().GetAuthConfig().Jwt.AccessTtl)*time.Minute, &claims)
	if err != nil {
		return "", nil, err
	}
	return token, &claims, nil
}

func (a *AuthService) ParseAccessToken(ctx context.Context, tokenString string) (*stores.AuthorizedClaims, error) {
	return a.authRepo.GetAccessTokenClaims(ctx, tokenString)
}

func (d *AuthService) IsReplayRequest(ctx context.Context, requestId, timestamp string) bool {
	timestampVal, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return true
	}

	maxInterval := app.GetGlobal().GetAuthConfig().ReplayMaxInterval
	if time.Now().Unix()-timestampVal > maxInterval {
		return true // 超过5分钟的请求视为无效
	}
	res, err := d.authRepo.SaveHandledRequest(ctx, requestId, time.Duration(maxInterval)*time.Second)
	if err != nil {
		return false
	}
	return !res
}

// 生成随机验证码
func (a *AuthService) GenerateImageCode() string {
	code := rand.Intn(9000) + 1000 // 生成4位数验证码
	return fmt.Sprintf("%04d", code)
}
