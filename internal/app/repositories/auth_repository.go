package repositories

import (
	"context"
	"goapp/internal/app/repositories/dao/query"
	"time"

	"github.com/sooomo/niu"
	"gorm.io/gorm"
)

const (
	KeyRevokedToken = "revoked_tokens"
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
