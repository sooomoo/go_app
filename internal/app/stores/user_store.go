package stores

import (
	"context"
	"goapp/internal/app/stores/dao/model"
	"goapp/internal/app/stores/dao/query"
	"goapp/pkg/cache"
	"goapp/pkg/core"
	"time"

	"gorm.io/datatypes"
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

type UserStore struct {
	cache *cache.Cache
}

func NewUserStore(cache *cache.Cache) *UserStore {
	return &UserStore{
		cache: cache,
	}
}

func (r *UserStore) Upsert(ctx context.Context, phone, ip string) (*model.User, error) {
	u := query.User

	// 使用事务进行 upsert
	err := query.Q.Transaction(func(tx *query.Query) error {
		_, err := tx.User.WithContext(ctx).Where(tx.User.Phone.Eq(phone)).Take()
		if err == gorm.ErrRecordNotFound {
			// 添加
			userId := core.NewID()
			err = tx.User.WithContext(ctx).Create(&model.User{
				ID:     userId,
				Phone:  phone,
				Name:   phone[3:6] + "****" + phone[10:],
				Role:   int32(RoleNormal),
				Status: UserStatusNormal,
				IP: core.SqlJSON{
					"init":   ip,
					"latest": ip,
				},
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			})
		} else {
			// 更新
			_, err = tx.User.WithContext(ctx).Where(tx.User.Phone.Eq(phone)).UpdateColumns(map[string]any{
				u.IP.ColumnName().String():        datatypes.JSONSet(u.IP.ColumnName().String()).Set("latest", ip),
				u.UpdatedAt.ColumnName().String(): time.Now().Unix(),
			})
		}

		return err
	})

	// err := u.WithContext(ctx).Clauses(clause.OnConflict{
	// 	Columns: []clause.Column{{Name: u.Phone.ColumnName().String()}},
	// 	DoUpdates: clause.Assignments(map[string]any{
	// 		u.UpdatedAt.ColumnName().String(): time.Now().Unix(),
	// 		u.IPLatest.ColumnName().String():  ip,
	// 	}),
	// }).Create(&model.User{
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
	if err != nil {
		return nil, err
	}

	return u.WithContext(ctx).Where(u.Phone.Eq(phone)).Take()
}

func (r *UserStore) GetById(ctx context.Context, userId int64) (*model.User, error) {
	return query.User.WithContext(ctx).Where(query.User.ID.Eq(userId)).Take()
}
