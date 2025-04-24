package service

import (
	"context"
	"fmt"
	"goapp/internal/app/global"
	"goapp/internal/app/repository"
	"goapp/internal/app/service/headers"
	"goapp/pkg/core"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type LoginRequest struct {
	Phone      string `json:"phone" binding:"required"`
	Code       string `json:"code" binding:"required"`
	SecureCode string `json:"secure_code" binding:"required"`
}

// AuthResponse JWT令牌对
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
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
		authRepo: repository.NewAuthRepository(global.Cache, global.Db),
		userRepo: repository.NewUserRepository(global.Cache, global.Db),
	}
}

func (a *AuthService) Authorize(ctx *gin.Context, req *LoginRequest) *AuthResponseDto {
	// 验证验证码
	if req.Code != "1234" {
		return &AuthResponseDto{Code: RespCodeInvalidMsgCode}
	}
	// 验证安全码
	if req.SecureCode != "8888" {
		return &AuthResponseDto{Code: RespCodeInvalidSecureCode}
	}

	// 通过手机号注册或获取用户信息
	ip := ctx.ClientIP()
	user, err := a.userRepo.Upsert(ctx, req.Phone, ip)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}
	// 该用户已被禁用
	if user.Status == repository.UserStatusBlock {
		ctx.AbortWithStatus(403)
		return nil
	}

	platform := headers.GetPlatform(ctx)
	ua := headers.GetUserAgentHashed(ctx)

	// 生成token
	accessToken, refreshToken, err := a.GenerateTokenPair(int(user.ID), ua, platform)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}

	clientId := core.NewUUIDWithoutDash()
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

	if platform == core.Web {
		a.setupAuthorizedCookie(ctx, accessToken, refreshToken)
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

	err := a.authRepo.DeleteRefreshToken(ctx, token)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}

	// 轮换 clientid 与 refresh token
	accessToken, refreshToken, err := a.GenerateTokenPair(int(credentials.UserId), ua, platform)
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
		a.setupAuthorizedCookie(ctx, accessToken, refreshToken)
		return &AuthResponseDto{Code: RespCodeSucceed}
	}

	return &AuthResponseDto{Code: RespCodeSucceed, Data: &AuthResponse{accessToken, refreshToken}}
}

func (a *AuthService) Logout(ctx *gin.Context) {
	accessToken := headers.GetAccessToken(ctx)
	refreshToken := headers.GetRefreshToken(ctx)

	a.RevokeAccessToken(ctx, accessToken)
	a.authRepo.DeleteRefreshToken(ctx, refreshToken)

	// 关闭所有的 hub
	// TODO: 可能不需要，客户端主动关闭也行

	// 删除 cookie
	jwtConfig := global.AppConfig.Authenticator.Jwt
	ctx.SetSameSite(http.SameSite(jwtConfig.CookieSameSiteMode))
	ctx.SetCookie(headers.CookieKeyAccessToken, "", -1000, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, true)
	ctx.SetCookie(headers.CookieKeyRefreshToken, "", -1000, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, true)
}

func (a *AuthService) setupAuthorizedCookie(ctx *gin.Context, accessToken, refreshToken string) {
	jwtConfig := global.AppConfig.Authenticator.Jwt
	ctx.SetSameSite(http.SameSite(jwtConfig.CookieSameSiteMode))
	atkMaxAge := int((time.Duration(jwtConfig.AccessTtl) * time.Minute).Seconds())
	rtkMaxAge := int((time.Duration(jwtConfig.RefreshTtl) * time.Minute).Seconds())
	ctx.SetCookie(headers.CookieKeyAccessToken, accessToken, atkMaxAge, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, true)
	ctx.SetCookie(headers.CookieKeyRefreshToken, refreshToken, rtkMaxAge, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, true)
}

func (a *AuthService) RevokeAccessToken(ctx context.Context, token string) error {
	if len(token) == 0 {
		return nil
	}
	expire := time.Duration(global.AppConfig.Authenticator.Jwt.AccessTtl) * time.Minute
	return a.authRepo.SaveRevokedToken(ctx, token, expire) // 调用Repository层的方法
}

func (a *AuthService) IsTokenRevoked(ctx context.Context, token string) (bool, error) {
	return a.authRepo.IsTokenRevoked(ctx, token) // 调用Repository层的方法
}

func (a *AuthService) GenerateTokenPair(userID int, userAgent string, platform core.Platform) (string, string, error) {
	accessToken, err := a.GenerateAccessToken(userID, userAgent, platform)
	if err != nil {
		return "", "", err
	}
	refreshToken := core.NewUUIDWithoutDash()
	return accessToken, refreshToken, nil
}

func (a *AuthService) GenerateAccessToken(userID int, userAgent string, platform core.Platform) (string, error) {
	jwtConfig := global.AppConfig.Authenticator.Jwt
	if len(jwtConfig.Secret) == 0 {
		panic("jwtSecret is empty")
	}
	claims := headers.AuthorizedClaims{
		UserId:    userID,
		Platform:  platform,
		UserAgent: userAgent,
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
func (a *AuthService) GenerateSmsCode() string {
	code := rand.Intn(9000) + 1000 // 生成4位数验证码
	return fmt.Sprintf("%04d", code)
}
