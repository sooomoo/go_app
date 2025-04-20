package service

import (
	"context"
	"fmt"
	"goapp/internal/app/global"
	"goapp/internal/app/repository"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sooomo/niu"
)

type RequestExtendData struct {
	Nonce     string
	Timestamp string
	Platform  niu.Platform
	Signature string
	SessionId string
}

type ClientKeys struct {
	SignPubKey []byte
	BoxPubKey  []byte
	ShareKey   []byte
}

type AuthorizedClaims struct {
	UserId               int          `json:"u"`
	Platform             niu.Platform `json:"p"`
	UserAgent            string       `json:"g"`
	jwt.RegisteredClaims              // 包含标准字段如 exp（过期时间）、iss（签发者）等
}

type LoginRequest struct {
	Phone      string `json:"phone" binding:"required"`
	Code       string `json:"code" binding:"required"`
	SecureCode string `json:"secure_code" binding:"required"`
}

// AuthResponse JWT令牌对
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ClientId     string `json:"client_id"`
}

type AuthResponseDto = ResponseDto[*AuthResponse]

const (
	KeyClaims     = "claims"
	KeyClientKeys = "client_keys"
	KeyExtendData = "extend_data"
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

	platform := a.GetPlatform(ctx)
	ua := a.GetUserAgentHashed(ctx)

	// 生成token
	accessToken, refreshToken, err := a.GenerateTokenPair(int(user.ID), ua, platform)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}

	clientId := niu.NewUUIDWithoutDash()
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

	if platform == niu.Web {
		a.setupAuthorizedCookie(ctx, accessToken, refreshToken, clientId)
		return &AuthResponseDto{Code: RespCodeSucceed}
	}

	return &AuthResponseDto{Code: RespCodeSucceed, Data: &AuthResponse{accessToken, refreshToken, clientId}}
}

func (a *AuthService) RefreshToken(ctx *gin.Context) *AuthResponseDto {
	token := a.GetRefreshToken(ctx)
	if len(token) == 0 {
		ctx.AbortWithStatus(401)
		return nil
	}
	revoked, err := a.IsTokenRevoked(ctx, token)
	if err != nil || revoked {
		ctx.AbortWithStatus(401)
		return nil
	}

	credentials, err := a.authRepo.GetRefreshTokenByValue(ctx, token)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}
	if credentials == nil {
		ctx.AbortWithStatus(401) // client need re-login
		return nil
	}
	ip := ctx.ClientIP()
	clientId := a.GetClientId(ctx)
	platform := a.GetPlatform(ctx)
	ua := a.GetUserAgentHashed(ctx)

	if clientId != credentials.ClientId || platform != credentials.Platform || ua != credentials.UserAgent {
		ctx.AbortWithStatus(401) // client need re-login
		return nil
	}

	err = a.RevokeToken(ctx, token, repository.TokenTypeRefresh)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}

	// 轮换 clientid 与 refresh token
	accessToken, refreshToken, err := a.GenerateTokenPair(int(credentials.UserId), ua, platform)
	if err != nil {
		return &AuthResponseDto{Code: RespCodeFailed, Msg: err.Error()}
	}
	clientId = niu.NewUUIDWithoutDash()
	// 将这些Token与该用户绑定
	err = a.authRepo.SaveRefreshToken(ctx, refreshToken, &repository.RefreshTokenCredentials{
		UserId:    credentials.UserId,
		Platform:  platform,
		Ip:        ip,
		UserAgent: ua,
		ClientId:  clientId,
	}, time.Duration(global.AppConfig.Authenticator.Jwt.RefreshTtl)*time.Minute)
	if err != nil {
		return &AuthResponseDto{Code: RespCodeFailed, Msg: err.Error()}
	}

	if platform == niu.Web {
		a.setupAuthorizedCookie(ctx, accessToken, refreshToken, clientId)
		return &AuthResponseDto{Code: RespCodeSucceed}
	}

	return &AuthResponseDto{Code: RespCodeSucceed, Data: &AuthResponse{accessToken, refreshToken, clientId}}
}

func (a *AuthService) setupAuthorizedCookie(ctx *gin.Context, accessToken, refreshToken, clientId string) {
	jwtConfig := global.AppConfig.Authenticator.Jwt
	ctx.SetSameSite(http.SameSite(jwtConfig.CookieSameSiteMode))
	atkMaxAge := int((time.Duration(jwtConfig.AccessTtl) * time.Minute).Seconds())
	rtkMaxAge := int((time.Duration(jwtConfig.RefreshTtl) * time.Minute).Seconds())
	ctx.SetCookie(jwtConfig.CookieAccessTokenKey, accessToken, atkMaxAge, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, true)
	ctx.SetCookie(jwtConfig.CookieRefreshTokenKey, refreshToken, rtkMaxAge, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, true)
	ctx.SetCookie("cli", clientId, rtkMaxAge, "/", jwtConfig.CookieDomain, jwtConfig.CookieSecure, true)
}

func (a *AuthService) GetPlatform(ctx *gin.Context) niu.Platform {
	extData, ok := ctx.Get(KeyExtendData)
	if ok {
		extendData, ok := extData.(*RequestExtendData)
		if ok {
			return extendData.Platform
		}
	}

	return niu.Unspecify
}

func (a *AuthService) GetAccessToken(ctx *gin.Context) string {
	// web单独处理
	if a.GetPlatform(ctx) == niu.Web {
		token, _ := ctx.Cookie(global.AppConfig.Authenticator.Jwt.CookieAccessTokenKey)
		return token
	}

	// 从请求头中获取令牌
	authHeader := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(authHeader[7:])
}

func (a *AuthService) GetRefreshToken(ctx *gin.Context) string {
	// web单独处理
	if a.GetPlatform(ctx) == niu.Web {
		token, _ := ctx.Cookie(global.AppConfig.Authenticator.Jwt.CookieRefreshTokenKey)
		return token
	}

	// 从请求头中获取令牌
	authHeader := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(authHeader[7:])
}

func (a *AuthService) GetClientId(ctx *gin.Context) string {
	// web单独处理
	if a.GetPlatform(ctx) == niu.Web {
		token, _ := ctx.Cookie("cli")
		return token
	}

	// 从请求头中获取令牌
	authHeader := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(authHeader[7:])
}

func (a *AuthService) GetUserAgentHashed(ctx *gin.Context) string {
	ua := ctx.Request.Header.Get("User-Agent")
	ua = strings.TrimSpace(ua)
	if len(ua) > 0 {
		ua = niu.HashSha256(ua)
	}
	return ua
}

func (a *AuthService) RevokeToken(ctx context.Context, token string, tokenType repository.TokenType) error {
	expire := time.Duration(global.AppConfig.Authenticator.Jwt.AccessTtl) * time.Minute
	if tokenType == repository.TokenTypeRefresh {
		expire = time.Duration(global.AppConfig.Authenticator.Jwt.RefreshTtl) * time.Minute
	}

	return a.authRepo.SaveRevokedToken(ctx, token, expire) // 调用Repository层的方法
}

func (a *AuthService) IsTokenRevoked(ctx context.Context, token string) (bool, error) {
	return a.authRepo.IsTokenRevoked(ctx, token) // 调用Repository层的方法
}

func (a *AuthService) GenerateTokenPair(userID int, userAgent string, platform niu.Platform) (string, string, error) {
	accessToken, err := a.GenerateAccessToken(userID, userAgent, platform)
	if err != nil {
		return "", "", err
	}
	refreshToken := niu.NewUUIDWithoutDash()
	return accessToken, refreshToken, nil
}

func (a *AuthService) GenerateAccessToken(userID int, userAgent string, platform niu.Platform) (string, error) {
	jwtConfig := global.AppConfig.Authenticator.Jwt
	if len(jwtConfig.Secret) == 0 {
		panic("jwtSecret is empty")
	}
	claims := AuthorizedClaims{
		UserId:    userID,
		Platform:  platform,
		UserAgent: userAgent,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(jwtConfig.AccessTtl) * time.Minute)), // 过期时间
			Issuer:    jwtConfig.Issuer,                                                                     // 签发者
			ID:        niu.NewUUIDWithoutDash(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtConfig.Secret)) // 使用 HMAC-SHA256 算法签名
}

func (a *AuthService) ParseAccessToken(tokenString string) (*AuthorizedClaims, error) {
	jwtConfig := global.AppConfig.Authenticator.Jwt
	if len(jwtConfig.Secret) == 0 {
		panic("jwtSecret is empty")
	}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&AuthorizedClaims{},
		func(token *jwt.Token) (any, error) {
			return []byte(jwtConfig.Secret), nil // 返回用于验证签名的密钥
		},
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*AuthorizedClaims); ok && token.Valid {
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

func (d *AuthService) GetClaims(c *gin.Context) *AuthorizedClaims {
	val, exist := c.Get(KeyClaims)
	if !exist {
		return nil
	}
	claims, ok := val.(*AuthorizedClaims)
	if !ok {
		return nil
	}
	return claims
}

func (a *AuthService) SaveClaims(ctx *gin.Context, claims *AuthorizedClaims) {
	ctx.Set(KeyClaims, claims)
}

// 生成随机验证码
func (a *AuthService) GenerateSmsCode() string {
	code := rand.Intn(9000) + 1000 // 生成4位数验证码
	return fmt.Sprintf("%04d", code)
}
