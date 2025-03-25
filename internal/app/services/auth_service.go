package services

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sooomo/niu"
)

type CustomClaims struct {
	UserId               int    `json:"u"`
	Role                 string `json:"r"`
	Platform             string `json:"p"`
	jwt.RegisteredClaims        // 包含标准字段如 exp（过期时间）、iss（签发者）等
}

type AuthService struct {
	Issuer        string
	TokenLiveTime time.Duration
	Secret        []byte
}

func NewAuthService() *AuthService {
	return &AuthService{"niu.com", 2 * time.Hour, []byte("anhdje2xksw3dksse")}
}

func (a *AuthService) AuthorizeRequest(ctx *gin.Context) int {
	platform := strings.TrimSpace(ctx.GetHeader("x-platform"))
	tokenString := strings.TrimPrefix(ctx.GetHeader("Authorization"), "Bearer ")
	if len(tokenString) == 0 {
		return http.StatusUnauthorized
	}

	// 解析Token
	claims, err := a.ParseToken(tokenString)
	if err != nil {
		return http.StatusUnauthorized
	}

	if len(platform) == 0 || claims.Platform != platform {
		return http.StatusBadRequest
	}

	// 检查token是否已经被吊销
	isRevoked, err := a.IsTokenRevoked(tokenString)
	if err != nil {
		return http.StatusInternalServerError
	}
	if isRevoked {
		return http.StatusInternalServerError
	}

	ctx.Set("user_id", claims.UserId) // 将用户 ID 注入上下文
	ctx.Set("role", claims.Role)
	ctx.Set("platform", platform)

	return 0
}

func (a *AuthService) IsReplayRequest(ctx *gin.Context) bool {

	// reqNonce := ctx.GetHeader(niu.HeaderSignNonce)
	return false
}

func (a *AuthService) ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&CustomClaims{},
		func(token *jwt.Token) (any, error) {
			return a.Secret, nil // 返回用于验证签名的密钥
		},
	)
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil // 验证通过后返回自定义声明数据
	}
	return nil, err
}

func (a *AuthService) GenerateToken(userID int, role string, platform niu.Platform) (string, error) {
	claims := CustomClaims{
		UserId:   userID,
		Role:     role,
		Platform: strconv.Itoa(int(platform)),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.TokenLiveTime)), // 过期时间
			Issuer:    a.Issuer,                                            // 签发者
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.Secret) // 使用 HMAC-SHA256 算法签名
}

func (a *AuthService) RevokeToken(tokenString string) error {
	return nil
}

func (a *AuthService) IsTokenRevoked(tokenString string) (bool, error) {
	return false, nil
}
