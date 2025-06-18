package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"goapp/pkg/cache"
	"goapp/pkg/core"
	"time"

	"github.com/redis/go-redis/v9"
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
}

func NewAuthRepository(cache *cache.Cache) *AuthRepository {
	return &AuthRepository{cache: cache}
}

func (a *AuthRepository) SaveCsrfToken(ctx context.Context, token, val string, expire time.Duration) error {
	_, err := a.cache.Set(ctx, fmt.Sprintf("csrf_token:%s", token), val, expire)
	return err
}
func (a *AuthRepository) GetCsrfToken(ctx context.Context, token string, del bool) (string, error) {
	if del {
		val, err := a.cache.GetDel(ctx, fmt.Sprintf("csrf_token:%s", token))
		if err == redis.Nil {
			return "", nil
		}
		return val, err
	} else {
		val, err := a.cache.Get(ctx, fmt.Sprintf("csrf_token:%s", token))
		if err == redis.Nil {
			return "", nil
		}
		return val, err
	}
}

func (a *AuthRepository) SaveHandledRequest(ctx context.Context, requestId string, expireAfter time.Duration) (bool, error) {
	exists, err := a.cache.SetNX(ctx, "handled_requests:"+requestId, "1", expireAfter)
	if err != nil {
		return false, err
	}
	return exists, nil
}

type AuthorizedClaims struct {
	UserId          int           `json:"userId"`
	Platform        core.Platform `json:"platform"`
	UserAgent       string        `json:"userAgent"`
	UserAgentHashed string        `json:"userAgentHashed"`
	ClientId        string        `json:"clientId"`
	Ip              string        `json:"ip"`
}

func (a *AuthRepository) SaveAccessToken(ctx context.Context, token string, ttl time.Duration, claims *AuthorizedClaims) error {
	key := fmt.Sprintf("access_token:%s", token)
	val, err := json.Marshal(claims)
	if err != nil {
		return err
	}

	_, err = a.cache.Set(ctx, key, val, ttl)
	return err
}

func (a *AuthRepository) DeleteAccessToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("access_token:%s", token)
	_, err := a.cache.KeyDel(ctx, key)
	return err
}

func (a *AuthRepository) GetAccessTokenClaims(ctx context.Context, token string) (*AuthorizedClaims, error) {
	if len(token) == 0 {
		return nil, redis.Nil
	}
	key := fmt.Sprintf("access_token:%s", token)
	val, err := a.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var dto AuthorizedClaims
	err = json.Unmarshal([]byte(val), &dto)
	if err != nil {
		return nil, err
	}

	return &dto, nil
}

func (a *AuthRepository) SaveRefreshToken(ctx context.Context, token string, credendials *AuthorizedClaims, expire time.Duration) error {
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

func (a *AuthRepository) GetRefreshTokenCredential(ctx context.Context, token string) *AuthorizedClaims {
	key := fmt.Sprintf("refresh_token:%s", token)
	jsonStr, err := a.cache.Get(ctx, key)
	if err != nil {
		return nil
	}

	var dto AuthorizedClaims
	err = json.Unmarshal([]byte(jsonStr), &dto)
	if err != nil {
		return nil
	}

	return &dto
}

func (a *AuthRepository) SaveSMSCode(ctx context.Context, phone string, code string) error {
	return nil
}
