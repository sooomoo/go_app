package service

import (
	"context"
	"errors"
	"fmt"
	"goapp/internal/app/global"
	"goapp/internal/app/repository"
	"goapp/internal/app/service/headers"
	"goapp/pkg/core"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mojocn/base64Captcha"
)

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

const (
	KeyClaims = "claims"
)

type AuthService struct {
	authRepo *repository.AuthRepository
	userRepo *repository.UserRepository
}

func NewAuthService() *AuthService {
	return &AuthService{
		authRepo: repository.NewAuthRepository(global.Cache),
		userRepo: repository.NewUserRepository(global.Cache),
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
	csrfToken := core.NewUUIDWithoutDash()
	// 将验证码存入缓存中
	dur := 10 * time.Minute
	err = a.authRepo.SaveCsrfToken(ctx, csrfToken, answer, dur)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}
	jwtConfig := global.AppConfig.Authenticator.Jwt
	ctx.SetSameSite(http.SameSite(jwtConfig.CookieSameSiteMode))
	ctx.SetCookie(headers.CookieKeyCsrfToken, csrfToken, int(dur.Seconds()), "/", "", jwtConfig.CookieSecure, true)
	// 返回验证码和csrf token
	return &PrepareLoginResponseDto{Code: RespCodeSucceed, Data: &PrepareLoginResponse{CsrfToken: csrfToken, ImageData: base64Str}}
}

var (
	countryCodeRegex = regexp.MustCompile(`^\d{3}$`)
	phoneRegex       = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

func (a *AuthService) Authorize(ctx *gin.Context, req *LoginRequest) *AuthResponseDto {
	csrfToken := headers.GetCsrfToken(ctx)
	if csrfToken != req.CsrfToken {
		ctx.AbortWithStatus(400)
		return nil
	}
	if !countryCodeRegex.MatchString(req.CountryCode) || !phoneRegex.MatchString(req.Phone) {
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
	if user.Status == repository.UserStatusBlock {
		ctx.AbortWithStatus(403)
		return nil
	}

	clientId := headers.GetClientId(ctx)
	platform := headers.GetPlatform(ctx)
	ua := headers.GetUserAgentHashed(ctx)

	// 生成token
	accessToken, refreshToken, err := a.GenerateTokenPair(int(user.ID), clientId, ua, platform)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}

	// 将这些Token与该用户绑定
	err = a.authRepo.SaveRefreshToken(ctx, refreshToken, &repository.RefreshTokenCredentials{
		UserId:    int(user.ID),
		Platform:  platform,
		Ip:        ip,
		UserAgent: ua,
		ClientId:  clientId,
	}, time.Duration(global.AppConfig.Authenticator.Jwt.RefreshTtl)*time.Minute)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}

	jwtConfig := global.AppConfig.Authenticator.Jwt
	ctx.SetSameSite(http.SameSite(jwtConfig.CookieSameSiteMode))
	ctx.SetCookie(headers.CookieKeyCsrfToken, "", -1000, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, true)

	if platform == core.Web {
		a.setupAuthorizedCookie(ctx, clientId, accessToken, refreshToken)
		return &AuthResponseDto{Code: RespCodeSucceed}
	}

	return &AuthResponseDto{Code: RespCodeSucceed, Data: &AuthResponse{accessToken, refreshToken}}
}

func (a *AuthService) RefreshToken(ctx *gin.Context) *AuthResponseDto {
	token := headers.GetRefreshToken(ctx)
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

	ip := ctx.ClientIP()
	clientId := headers.GetClientId(ctx)
	platform := headers.GetPlatform(ctx)
	ua := headers.GetUserAgentHashed(ctx)
	if clientId != credentials.ClientId || platform != credentials.Platform || ua != credentials.UserAgent {
		ctx.AbortWithStatus(401) // client need re-login
		return nil
	}

	err := a.DeleteRefreshToken(ctx, token)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}

	// 轮换 clientid 与 refresh token
	accessToken, refreshToken, err := a.GenerateTokenPair(int(credentials.UserId), clientId, ua, platform)
	if err != nil {
		ctx.AbortWithStatus(401) // token 已经删除，此时只能重新登录
		return nil
	}

	err = a.authRepo.SaveRefreshToken(ctx, refreshToken, &repository.RefreshTokenCredentials{
		UserId:    credentials.UserId,
		Platform:  platform,
		Ip:        ip,
		UserAgent: ua,
		ClientId:  clientId,
	}, time.Duration(global.AppConfig.Authenticator.Jwt.RefreshTtl)*time.Minute)
	if err != nil {
		ctx.AbortWithStatus(401) // token 已经删除，此时只能重新登录
		return nil
	}

	if platform == core.Web {
		a.setupAuthorizedCookie(ctx, clientId, accessToken, refreshToken)
		return &AuthResponseDto{Code: RespCodeSucceed}
	}

	return &AuthResponseDto{Code: RespCodeSucceed, Data: &AuthResponse{accessToken, refreshToken}}
}

func (a *AuthService) Logout(ctx *gin.Context) {
	accessToken := headers.GetAccessToken(ctx)
	refreshToken := headers.GetRefreshToken(ctx)

	a.RevokeAccessToken(ctx, accessToken)
	a.DeleteRefreshToken(ctx, refreshToken)

	// 关闭所有的 hub
	// TODO: 可能不需要，客户端主动关闭也行

	// 删除 cookie
	jwtConfig := global.AppConfig.Authenticator.Jwt
	ctx.SetSameSite(http.SameSite(jwtConfig.CookieSameSiteMode))
	ctx.SetCookie(headers.CookieKeyAccessToken, "", -1000, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, true)
	ctx.SetCookie(headers.CookieKeyRefreshToken, "", -1000, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, true)
}

func (a *AuthService) setupAuthorizedCookie(ctx *gin.Context, clientId, accessToken, refreshToken string) {
	jwtConfig := global.AppConfig.Authenticator.Jwt
	ctx.SetSameSite(http.SameSite(jwtConfig.CookieSameSiteMode))
	atkMaxAge := int((time.Duration(jwtConfig.AccessTtl) * time.Minute).Seconds())
	rtkMaxAge := int((time.Duration(jwtConfig.RefreshTtl) * time.Minute).Seconds())
	cliMaxAge := rtkMaxAge + int((time.Hour * 24 * 30).Seconds())
	ctx.SetCookie(headers.CookieKeyAccessToken, accessToken, atkMaxAge, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, true)
	ctx.SetCookie(headers.CookieKeyRefreshToken, refreshToken, rtkMaxAge, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, true)
	ctx.SetCookie(headers.CookieKeyClientId, clientId, cliMaxAge, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, true)
}

func (a *AuthService) IsClaimsValid(ctx *gin.Context, claims *headers.AuthorizedClaims) bool {
	clientId := headers.GetClientId(ctx)
	platform := headers.GetPlatform(ctx)
	ua := headers.GetUserAgentHashed(ctx)
	return clientId == claims.ClientId && platform == claims.Platform && ua == claims.UserAgent
}

func (a *AuthService) RevokeAccessToken(ctx context.Context, token string) error {
	if len(token) == 0 {
		return nil
	}
	expire := time.Duration(global.AppConfig.Authenticator.Jwt.AccessTtl) * time.Minute
	return a.authRepo.SaveRevokedToken(ctx, token, expire) // 调用Repository层的方法
}

func (a *AuthService) DeleteRefreshToken(ctx context.Context, refreshToken string) error {
	if len(refreshToken) == 0 {
		return nil
	}
	return a.authRepo.DeleteRefreshToken(ctx, refreshToken) // 调用Repository层的方法
}

func (a *AuthService) IsTokenRevoked(ctx context.Context, token string) (bool, error) {
	return a.authRepo.IsTokenRevoked(ctx, token) // 调用Repository层的方法
}

func (a *AuthService) GenerateTokenPair(userID int, clientId, userAgent string, platform core.Platform) (string, string, error) {
	accessToken, err := a.GenerateAccessToken(userID, clientId, userAgent, platform)
	if err != nil {
		return "", "", err
	}
	refreshToken := core.NewUUIDWithoutDash()
	return accessToken, refreshToken, nil
}

func (a *AuthService) GenerateAccessToken(userID int, clientId, userAgent string, platform core.Platform) (string, error) {
	jwtConfig := global.AppConfig.Authenticator.Jwt
	if len(jwtConfig.Secret) == 0 {
		panic("jwtSecret is empty")
	}
	if len(clientId) == 0 || len(userAgent) == 0 || platform == core.Unspecify {
		return "", errors.New("invalid args")
	}
	claims := headers.AuthorizedClaims{
		UserId:    userID,
		Platform:  platform,
		UserAgent: userAgent,
		ClientId:  clientId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(jwtConfig.AccessTtl) * time.Minute)), // 过期时间
			Issuer:    jwtConfig.Issuer,                                                                     // 签发者
			ID:        core.NewUUIDWithoutDash(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtConfig.Secret)) // 使用 HMAC-SHA256 算法签名
}

func (a *AuthService) ParseAccessToken(tokenString string) (*headers.AuthorizedClaims, error) {
	jwtConfig := global.AppConfig.Authenticator.Jwt
	if len(jwtConfig.Secret) == 0 {
		panic("jwtSecret is empty")
	}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&headers.AuthorizedClaims{},
		func(token *jwt.Token) (any, error) {
			return []byte(jwtConfig.Secret), nil // 返回用于验证签名的密钥
		},
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*headers.AuthorizedClaims); ok && token.Valid {
		return claims, nil // 验证通过后返回自定义声明数据
	}
	return nil, err
}

func (d *AuthService) IsReplayRequest(ctx context.Context, requestId, timestamp string) bool {
	timestampVal, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return true
	}

	maxInterval := global.AppConfig.Authenticator.ReplayMaxInterval
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
