package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"goapp/internal/app/repository/dao/query"
	"goapp/pkg/cache"
	"goapp/pkg/core"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	KeyRevokedToken = "revoked_tokens"
)

type TokenType int32

const (
	TokenTypeAccess  TokenType = 1
	TokenTypeRefresh TokenType = 2
)

type AuthRepository struct {
	cache *cache.Cache
	db    *gorm.DB
	query *query.Query
}

func NewAuthRepository(cache *cache.Cache, db *gorm.DB) *AuthRepository {
	return &AuthRepository{
		cache: cache,
		db:    db,
		query: query.Use(db),
	}
}

func (a *AuthRepository) SaveRevokedToken(ctx context.Context, token string, expire time.Duration) error {
	// 将Token添加到Redis中,过期时间为token的最大有效时间（比如两小时）
	// 因为Token在使用时，会验证其有效期
	_, err := a.cache.Set(ctx, fmt.Sprintf("revoked_token:%s", token), "1", expire)
	return err
}

func (a *AuthRepository) IsTokenRevoked(ctx context.Context, token string) (bool, error) {
	val, err := a.cache.Get(ctx, fmt.Sprintf("revoked_token:%s", token))
	if err == redis.Nil {
		return false, nil
	}
	return val == "1", err
}

func (a *AuthRepository) SaveHandledRequest(ctx context.Context, requestId string, expireAfter time.Duration) (bool, error) {
	exists, err := a.cache.SetNX(ctx, "handled_requests:"+requestId, "1", expireAfter)
	if err != nil {
		return false, err
	}
	return exists, nil
}

type RefreshTokenCredentials struct {
	UserId    int           `json:"user_id"`
	Platform  core.Platform `json:"platform"`
	ClientId  string        `json:"client_id"`
	UserAgent string        `json:"user_agent"`
	Ip        string        `json:"ip"`
}

func (a *AuthRepository) SaveRefreshToken(ctx context.Context, token string, credendials *RefreshTokenCredentials, expire time.Duration) error {
	key := fmt.Sprintf("refresh_token:%s", token)
	val, err := json.Marshal(credendials)
	if err != nil {
		return err
	}

	_, err = a.cache.Set(ctx, key, val, expire)
	return err
}

func (a *AuthRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("refresh_token:%s", token)
	_, err := a.cache.KeyDel(ctx, key)
	return err
}

func (a *AuthRepository) GetRefreshTokenByValue(ctx context.Context, token string) *RefreshTokenCredentials {
	key := fmt.Sprintf("refresh_token:%s", token)
	jsonStr, err := a.cache.Get(ctx, key)
	if err != nil {
		return nil
	}

	var dto RefreshTokenCredentials
	err = json.Unmarshal([]byte(jsonStr), &dto)
	if err != nil {
		return nil
	}

	return &dto
}

func (a *AuthRepository) SaveSMSCode(ctx context.Context, phone string, code string) error {
	return nil
}
