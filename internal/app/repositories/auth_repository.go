package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"goapp/internal/app/repositories/dao/model"
	"goapp/internal/app/repositories/dao/query"
	"time"

	"github.com/sooomo/niu"
	"gorm.io/gorm"
)

const (
	KeyRevokedToken = "revoked_tokens"
)

const (
	TokenTypeAccess  int32 = 1
	TokenTypeRefresh int32 = 2
)

type RepositoryAuth struct {
	cache *niu.Cache
	db    *gorm.DB
	query *query.Query
}

func NewRepositoryAuth(cache *niu.Cache, db *gorm.DB) *RepositoryAuth {
	return &RepositoryAuth{
		cache: cache,
		db:    db,
		query: query.Use(db),
	}
}

func (a *RepositoryAuth) SaveRevokedToken(ctx context.Context, token string) error {
	// 将Token添加到Redis集合中，表示已吊销
	_, err := a.cache.SAdd(ctx, KeyRevokedToken, token)
	return err
}

func (a *RepositoryAuth) IsTokenRevoked(ctx context.Context, token string) (bool, error) {
	// 检查Token是否存在于Redis集合中
	exists, err := a.cache.SIsMember(ctx, KeyRevokedToken, token)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (a *RepositoryAuth) SaveHandledRequest(ctx context.Context, requestId string) (bool, error) {
	exists, err := a.cache.SetNX(ctx, "handled_requests:"+requestId, "1", time.Duration(300)*time.Second)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (a *RepositoryAuth) SaveBindings(
	ctx context.Context, userId int64, platform niu.Platform, ip,
	accessToken, refreshToken string,
	accessExpire, refreshExpire int64) error {
	accessDto := &model.UserToken{
		UserID:    userId,
		Platform:  int32(platform),
		Type:      TokenTypeAccess,
		Token:     accessToken,
		IP:        ip,
		CreatedAt: time.Now().Unix(),
		ExpireAt:  accessExpire,
	}
	refreshDto := &model.UserToken{
		UserID:    userId,
		Platform:  int32(platform),
		Type:      TokenTypeRefresh,
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

func (a *RepositoryAuth) GetRefreshTokenByValue(ctx context.Context, token string) (*model.UserToken, error) {
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
