package services

import (
	"context"
	"errors"
	"goapp/internal/app/global"
	"goapp/internal/app/repositories"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sooomo/niu"
)

type AuthorizedClaims struct {
	UserId               int          `json:"u"`
	Roles                []string     `json:"r"`
	Platform             niu.Platform `json:"p"`
	Type                 string       `json:"t"` // 类型，如：access_token, refresh_token, message_token, etc.
	jwt.RegisteredClaims              // 包含标准字段如 exp（过期时间）、iss（签发者）等
}

type LoginRequest struct {
	Phone      string `json:"phone" binding:"required"`
	Code       string `json:"code" binding:"required"`
	SecureCode string `json:"secure_code" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenRequest struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

const (
	KeyClaims = "claims"
)

type AuthService struct {
	authRepo *repositories.RepositoryAuth
	userRepo *repositories.RepositoryUser
}

func NewAuthService() *AuthService {
	return &AuthService{
		authRepo: repositories.NewRepositoryAuth(global.Cache, global.Db),
		userRepo: repositories.NewRepositoryUser(global.Cache, global.Db),
	}
}

func (s *AuthService) Authorize(ctx *gin.Context, req *LoginRequest, platform niu.Platform) *niu.ReplyDto[ReplyCode, LoginResponse] {
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
	user, err := s.userRepo.Upsert(ctx, req.Phone)
	if err != nil {
		reply.Code = ReplyCodeFailed
		reply.Msg = err.Error()
		return reply
	}

	// 生成token
	accessToken, err := s.GenerateAccessToken(int(user.ID), strings.Split(user.Roles, ","), platform)
	if err != nil {
		reply.Code = ReplyCodeFailed
		reply.Msg = err.Error()
		return reply
	}
	refreshToken, err := s.GenerateRefreshToken(int(user.ID), strings.Split(user.Roles, ","), platform)
	if err != nil {
		reply.Code = ReplyCodeFailed
		reply.Msg = err.Error()
		return reply
	}

	// 将这些Token与该用户绑定
	jwtConfig := global.AppConfig.Authenticator.Jwt
	err = s.authRepo.SaveBindings(ctx, user.ID, platform, ctx.ClientIP(), accessToken, refreshToken,
		time.Now().Add(time.Duration(jwtConfig.AccessTtl)).Unix(),
		time.Now().Add(time.Duration(jwtConfig.RefreshTtl)).Unix())
	if err != nil {
		reply.Code = ReplyCodeFailed
		reply.Msg = err.Error()
		return reply
	}

	reply.Code = ReplyCodeSucceed
	reply.Data = LoginResponse{accessToken, refreshToken}
	return reply
}

func (s *AuthService) RefreshToken(ctx *gin.Context, req *RefreshTokenRequest) *niu.ReplyDto[ReplyCode, LoginResponse] {
	authHeader := s.GetAuthorizationHeader(ctx)
	if req.RefreshToken != authHeader {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return nil
	}
	userToken, err := s.authRepo.GetRefreshTokenByValue(ctx, authHeader)
	if err != nil {
		ctx.AbortWithError(500, err)
		return nil
	}
	claims := s.GetClaims(ctx)
	if userToken == nil || claims == nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("token not exist"))
		return nil
	}

	reply := &niu.ReplyDto[ReplyCode, LoginResponse]{}
	// 生成token
	accessToken, err := s.GenerateAccessToken(int(claims.UserId), claims.Roles, claims.Platform)
	if err != nil {
		reply.Code = ReplyCodeFailed
		reply.Msg = err.Error()
		return reply
	}
	refreshToken, err := s.GenerateRefreshToken(int(claims.UserId), claims.Roles, claims.Platform)
	if err != nil {
		reply.Code = ReplyCodeFailed
		reply.Msg = err.Error()
		return reply
	}

	// 将这些Token与该用户绑定
	jwtConfig := global.AppConfig.Authenticator.Jwt
	err = s.authRepo.SaveBindings(ctx, int64(claims.UserId), claims.Platform, ctx.ClientIP(), accessToken, refreshToken,
		time.Now().Add(time.Duration(jwtConfig.AccessTtl)).Unix(),
		time.Now().Add(time.Duration(jwtConfig.RefreshTtl)).Unix())
	if err != nil {
		reply.Code = ReplyCodeFailed
		reply.Msg = err.Error()
		return reply
	}

	reply.Code = ReplyCodeSucceed
	reply.Data = LoginResponse{accessToken, refreshToken}
	return reply
}

func (s *AuthService) GetAuthorizationHeader(ctx *gin.Context) string {
	return strings.TrimSpace(strings.TrimPrefix(ctx.GetHeader("Authorization"), "Bearer "))
}

func (a *AuthService) RevokeToken(ctx context.Context, token string) error {
	return a.authRepo.SaveRevokedToken(ctx, token) // 调用Repository层的方法
}

func (a *AuthService) IsTokenRevoked(ctx context.Context, token string) (bool, error) {
	return a.authRepo.IsTokenRevoked(ctx, token) // 调用Repository层的方法
}

func (a *AuthService) GenerateAccessToken(userID int, roles []string, platform niu.Platform) (string, error) {
	jwtConfig := global.AppConfig.Authenticator.Jwt
	if len(jwtConfig.Secret) == 0 {
		panic("jwtSecret is empty")
	}
	claims := AuthorizedClaims{
		UserId:   userID,
		Roles:    roles,
		Platform: platform,
		Type:     "a",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(jwtConfig.AccessTtl) * time.Minute)), // 过期时间
			Issuer:    jwtConfig.Issuer,                                                                     // 签发者
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtConfig.Secret)) // 使用 HMAC-SHA256 算法签名
}

func (a *AuthService) GenerateRefreshToken(userID int, roles []string, platform niu.Platform) (string, error) {
	jwtConfig := global.AppConfig.Authenticator.Jwt
	if len(jwtConfig.Secret) == 0 {
		panic("jwtSecret is empty")
	}
	claims := AuthorizedClaims{
		UserId:   userID,
		Platform: platform,
		Roles:    roles,
		Type:     "r",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(jwtConfig.RefreshTtl) * time.Minute)), // 过期时间
			Issuer:    jwtConfig.Issuer,                                                                      // 签发者
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtConfig.Secret)) // 使用 HMAC-SHA256 算法签名
}

func (a *AuthService) ParseToken(tokenString string) (*AuthorizedClaims, error) {
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
	)
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*AuthorizedClaims); ok && token.Valid {
		return claims, nil // 验证通过后返回自定义声明数据
	}
	return nil, err
}

func (d *AuthService) IsReplayRequest(ctx context.Context, nonce, timestamp string) bool {
	timestampVal, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return true
	}
	if time.Now().Unix()-timestampVal > 300 {
		return true // 超过5分钟的请求视为无效
	}
	res, err := d.authRepo.SaveHandledRequest(ctx, nonce)
	if err != nil {
		return false
	}
	return !res
}

func (d *AuthService) GetPlatform(c *gin.Context) string {
	return strings.TrimSpace(c.GetHeader("X-Platform"))
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

func (a *AuthService) GetSigner(ctx *gin.Context) (niu.Signer, error) {
	// TODO:
	return nil, nil
}

func (a *AuthService) GetCryptor(ctx *gin.Context) (niu.Cryptor, error) {
	// TODO:
	return nil, nil
}
