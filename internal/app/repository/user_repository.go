package repository

import (
	"context"
	"goapp/internal/app/repository/dao/model"
	"goapp/internal/app/repository/dao/query"
	"time"

	"github.com/sooomo/niu"
	"gorm.io/gorm"
)

const (
	UserStatusNormal int32 = 1
	UserStatusBlock  int32 = 2
)

type UserRepository struct {
	cache *niu.Cache
	db    *gorm.DB
	query *query.Query
}

func NewUserRepository(cache *niu.Cache, db *gorm.DB) *UserRepository {
	return &UserRepository{
		cache: cache,
		db:    db,
		query: query.Use(db),
	}
}

func (r *UserRepository) Upsert(ctx context.Context, phone string) (*model.User, error) {
	err := r.query.User.WithContext(ctx).WriteDB().Save(&model.User{
		Phone:     phone,
		Name:      phone[:4] + "****" + phone[8:],
		Status:    UserStatusNormal,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	})
	if err != nil {
		return nil, err
	}
	return r.query.User.WithContext(ctx).ReadDB().Where(r.query.User.Phone.Eq(phone)).First()
}

func (r *UserRepository) GetById(ctx context.Context, userId int64) (*model.User, error) {
	return r.query.User.WithContext(ctx).ReadDB().Where(r.query.User.ID.Eq(userId)).First()
}
