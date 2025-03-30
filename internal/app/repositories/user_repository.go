package repositories

import (
	"context"
	"goapp/internal/app/repositories/dao/model"
	"goapp/internal/app/repositories/dao/query"

	"github.com/sooomo/niu"
	"gorm.io/gorm"
)

type RepositoryOfUser struct {
	cache *niu.Cache
	db    *gorm.DB
	query *query.Query
}

func NewRepositoryOfUser(cache *niu.Cache, db *gorm.DB) *RepositoryOfUser {
	return &RepositoryOfUser{
		cache: cache,
		db:    db,
		query: query.Use(db),
	}
}

func (r *RepositoryOfUser) Upsert(ctx context.Context, phone string) (*model.User, error) {
	err := r.query.User.WithContext(ctx).WriteDB().Save(&model.User{Phone: phone})
	if err != nil {
		return nil, err
	}
	return r.query.User.WithContext(ctx).ReadDB().Where(r.query.User.Phone.Eq(phone)).First()
}

func (r *RepositoryOfUser) GetById(ctx context.Context, userId int64) (*model.User, error) {
	return r.query.User.WithContext(ctx).ReadDB().Where(r.query.User.ID.Eq(userId)).First()
}
