package repository

import (
	"context"
	"goapp/internal/app/global"
	"goapp/internal/app/repository/dao/model"
	"goapp/internal/app/repository/dao/query"
	"time"

	"github.com/sooomo/niu"
	"gorm.io/gen/field"
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

func (r *UserRepository) Upsert(ctx context.Context, phone, ip string) (*model.User, error) {
	userId, err := global.UserIdGenerator.Next(ctx)
	if err != nil {
		return nil, err
	}

	// var ret model.User
	// err = r.query.User.WithContext(ctx).Clauses(clause.OnConflict{
	// 	Columns: []clause.Column{{Name: r.query.User.Phone.ColumnName().String()}},
	// 	DoUpdates: clause.Assignments(map[string]any{
	// 		r.query.User.UpdatedAt.ColumnName().String(): time.Now().Unix(),
	// 		r.query.User.IPLatest.ColumnName().String():  ip,
	// 	}),
	// }).DO.Returning(&ret).Create(&model.User{
	// 	ID:        int64(userId),
	// 	Phone:     phone,
	// 	Name:      phone[3:6] + "****" + phone[10:],
	// 	Role:      int32(RoleNormal),
	// 	Status:    UserStatusNormal,
	// 	IPInit:    ip,
	// 	IPLatest:  ip,
	// 	CreatedAt: time.Now().Unix(),
	// 	UpdatedAt: time.Now().Unix(),
	// })
	ret, err := r.query.User.WithContext(ctx).Assign(field.Attrs(&model.User{
		IPLatest:  ip,
		UpdatedAt: time.Now().Unix(),
	})).Attrs(field.Attrs(&model.User{
		ID:        int64(userId),
		Phone:     phone,
		Name:      phone[3:6] + "****" + phone[10:],
		Role:      int32(RoleNormal),
		Status:    UserStatusNormal,
		IPInit:    ip,
		IPLatest:  ip,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	})).Where(r.query.User.Phone.Eq(phone)).FirstOrCreate()
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (r *UserRepository) GetById(ctx context.Context, userId int64) (*model.User, error) {
	return r.query.User.WithContext(ctx).ReadDB().Where(r.query.User.ID.Eq(userId)).First()
}
