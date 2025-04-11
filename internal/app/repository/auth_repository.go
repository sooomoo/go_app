package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"goapp/internal/app/repository/dao/model"
	"goapp/internal/app/repository/dao/query"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sooomo/niu"
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
	cache *niu.Cache
	db    *gorm.DB
	query *query.Query
}

func NewAuthRepository(cache *niu.Cache, db *gorm.DB) *AuthRepository {
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

func (a *AuthRepository) SaveBindings(
	ctx context.Context, userId int64, platform niu.Platform, ip,
	accessToken, refreshToken string,
	accessExpire, refreshExpire int64) error {
	accessDto := &model.UserToken{
		UserID:    userId,
		Platform:  int32(platform),
		Type:      int32(TokenTypeAccess),
		Token:     accessToken,
		IP:        ip,
		CreatedAt: time.Now().Unix(),
		ExpireAt:  accessExpire,
	}
	refreshDto := &model.UserToken{
		UserID:    userId,
		Platform:  int32(platform),
		Type:      int32(TokenTypeRefresh),
		Token:     refreshToken,
		IP:        ip,
		CreatedAt: time.Now().Unix(),
		ExpireAt:  refreshExpire,
	}
	err := a.query.UserToken.WithContext(ctx).WriteDB().Save(accessDto, refreshDto)
	if err != nil {
		return err
	}

	dur := time.Now().Sub(time.Unix(refreshDto.ExpireAt, 0))
	refreshDtoJson, err := json.Marshal(refreshDto)
	if err != nil {
		a.cache.Set(ctx, fmt.Sprintf("refresh_token:%s", refreshToken), string(refreshDtoJson), dur)
	}

	return nil
}

func (a *AuthRepository) GetRefreshTokenByValue(ctx context.Context, token string) (*model.UserToken, error) {
	key := fmt.Sprintf("refresh_token:%s", token)
	jsonStr, err := a.cache.Get(ctx, key)
	if err == nil {
		var dto model.UserToken
		if err = json.Unmarshal([]byte(jsonStr), &dto); err == nil {
			return &dto, nil
		}
	}

	return a.query.UserToken.WithContext(ctx).ReadDB().Where(a.query.UserToken.Token.Eq(token)).First()
}

func (a *AuthRepository) SaveSMSCode(ctx context.Context, phone string, code string) error {
	return nil
}
