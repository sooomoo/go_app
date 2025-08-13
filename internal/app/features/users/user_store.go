package users

import (
	"context"
	"goapp/internal/app/dao/model"
	"goapp/internal/app/dao/query"
	"goapp/internal/app/global"
	"goapp/pkg/cache"
	"goapp/pkg/ids"
	"time"

	"gorm.io/gorm/clause"
)

const (
	UserStatusNormal int16 = 0
	UserStatusBlock  int16 = 1
)

type Role int32

const (
	RoleNormal Role = 0b00000000 // 普通用户
	RolePro    Role = 0b00000001 // 普通用户
	RoleAdmin  Role = 0b10000000 // 管理员
)

const (
	CacheKeyPrefixUser     = "user:"
	CacheKeyPrefixLatestIP = "latestip:"
)

type UserStore struct {
	cache *cache.Cache
}

func NewUserStore() *UserStore {
	return &UserStore{
		cache: global.GetCache(),
	}
}

func (r *UserStore) Upsert(ctx context.Context, phone, ip string) (*model.User, error) {
	u := query.User

	// // 使用事务进行 upsert
	// err := query.Q.Transaction(func(tx *query.Query) error {
	// 	_, err := tx.User.WithContext(ctx).Where(tx.User.Phone.Eq(phone)).Take()
	// 	if err == gorm.ErrRecordNotFound {
	// 		// 添加
	// 		userId := ids.NewUID()
	// 		err = tx.User.WithContext(ctx).Create(&model.User{
	// 			ID:     userId,
	// 			Phone:  phone,
	// 			Name:   phone[3:6] + "****" + phone[10:],
	// 			Role:   int32(RoleNormal),
	// 			Status: UserStatusNormal,
	// 			IP: db.JSON{
	// 				"init":   ip,
	// 				"latest": ip,
	// 			},
	// 			CreatedAt: time.Now().Unix(),
	// 			UpdatedAt: time.Now().Unix(),
	// 		})
	// 	} else {
	// 		// 更新
	// 		_, err = tx.User.WithContext(ctx).Where(tx.User.Phone.Eq(phone)).UpdateColumns(map[string]any{
	// 			u.IP.ColumnName().String():        datatypes.JSONSet(u.IP.ColumnName().String()).Set("{latest}", ip),
	// 			u.UpdatedAt.ColumnName().String(): time.Now().Unix(),
	// 		})
	// 	}

	// 	return err
	// })

	err := u.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: u.Phone.ColumnName().String()}},
		DoNothing: true,
	}).Create(&model.User{
		ID:        ids.NewUID(),
		Phone:     phone,
		Name:      phone[3:6] + "****" + phone[10:],
		Role:      int32(RoleNormal),
		Status:    UserStatusNormal,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	})
	if err != nil {
		return nil, err
	}
	user, err := u.WithContext(ctx).Where(u.Phone.Eq(phone)).Take()
	if err != nil {
		return nil, err
	}

	r.cache.SetJson(ctx, CacheKeyPrefixUser+user.ID.String(), user, time.Hour)
	r.UpsertLatestIP(ctx, user.ID, ip)

	return user, nil
}

func (r *UserStore) GetById(ctx context.Context, userId ids.UID) (*model.User, error) {
	key := CacheKeyPrefixUser + userId.String()
	var user model.User
	err := r.cache.GetJson(ctx, key, &user)
	if err != nil {
		user, err := query.User.WithContext(ctx).Where(query.User.ID.Eq(userId)).Take()
		if err != nil {
			return nil, err
		}
		r.cache.SetJson(ctx, key, user, time.Hour)
		return user, nil
	}
	return &user, nil
}

func (r *UserStore) UpsertLatestIP(ctx context.Context, userID ids.UID, ip string) error {
	uip := query.UserIP
	err := uip.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: uip.ID.ColumnName().String()}},
		DoUpdates: clause.Assignments(map[string]any{
			uip.UpdatedAt.ColumnName().String(): time.Now().Unix(),
			uip.Latest.ColumnName().String():    ip,
		}),
	}).Create(&model.UserIP{
		ID:        userID,
		Register:  ip,
		Latest:    ip,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	})
	if err != nil {
		r.cache.Set(ctx, CacheKeyPrefixLatestIP+userID.String(), ip, time.Hour)
	}
	return err
}

func (r *UserStore) GetLatestIP(ctx context.Context, userId ids.UID) string {
	key := CacheKeyPrefixLatestIP + userId.String()
	ip, _ := r.cache.Get(ctx, key)
	if len(ip) != 0 {
		return ip
	}

	uipv, err := query.UserIP.WithContext(ctx).Select(query.UserIP.Latest).Where(query.UserIP.ID.Eq(userId)).Take()
	if err != nil {
		return ""
	}
	r.cache.Set(ctx, key, uipv.Latest, time.Hour)
	return uipv.Latest
}
