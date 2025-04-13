package repository

import (
	"context"
	"goapp/internal/app/global"
	"goapp/internal/app/repository/dao/model"
	"goapp/internal/app/repository/dao/query"
	"time"

	"github.com/sooomo/niu"
	"gorm.io/gorm"
)

const (
	UserStatusNormal int32 = 0
	UserStatusBlock  int32 = 1
)

type Role int32

const (
	RoleNormal Role = 0b00000000 // 普通用户
	RolePro    Role = 0b00000001 // 普通用户
	RoleAdmin  Role = 0b10000000 // 管理员
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

	err := r.query.Transaction(func(tx *query.Query) error {
		u, err := tx.User.WithContext(ctx).Where(tx.User.Phone.Eq(phone)).First()
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err == gorm.ErrRecordNotFound {
			userId, err := global.UserIdGenerator.Next(ctx)
			if err != nil {
				return err
			}
			// add new one
			u = &model.User{
				ID:        int64(userId),
				Phone:     phone,
				Name:      phone[3:6] + "****" + phone[10:],
				Role:      int32(RoleNormal),
				Status:    UserStatusNormal,
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			}

			err = tx.User.WithContext(ctx).Save(u)
			if err != nil {
				return err
			}
		} else {
			// update
			res, err := tx.User.WithContext(ctx).Where(tx.User.ID.Eq(u.ID)).Update(tx.User.UpdatedAt, time.Now().Unix())
			if err != nil {
				return err
			}
			if res.RowsAffected < 1 {
				// return errors.New("update fail")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.query.User.WithContext(ctx).ReadDB().Where(r.query.User.Phone.Eq(phone)).First()
}

func (r *UserRepository) GetById(ctx context.Context, userId int64) (*model.User, error) {
	return r.query.User.WithContext(ctx).ReadDB().Where(r.query.User.ID.Eq(userId)).First()
}
