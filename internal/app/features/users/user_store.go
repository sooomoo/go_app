package users

import (
	"context"
	"goapp/internal/app/global"
	"goapp/internal/app/models"
	"goapp/pkg/cache"
	"goapp/pkg/db"
	"goapp/pkg/ids"
	"time"

	"github.com/jackc/pgerrcode"
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

func (r *UserStore) Upsert(ctx context.Context, phone, ip string) (*models.User, error) {
	user := &models.User{
		ID:     ids.NewUID(),
		Phone:  phone,
		Name:   phone[3:6] + "****" + phone[10:],
		Role:   models.UserRoleNormal,
		Status: models.UserStatusNormal,
	}

	// 方式一：通过唯一索引的错误来判断是否存在
	_, err := global.DB().NewInsert().Model(user).
		Exec(ctx)
	if err != nil && !db.IsPGErrorCode(err, pgerrcode.UniqueViolation) {
		return nil, err
	}

	// // 方式二：通过约束来处理唯一索引冲突
	// ALTER TABLE users DROP constraint if Exists users_unique;
	// ALTER TABLE users
	// ADD CONSTRAINT users_unique
	// UNIQUE NULLS NOT DISTINCT (phone, deleted_at);
	// _, err := global.DB().NewInsert().Model(user).
	// 	On("CONFLICT on CONSTRAINT users_unique DO NOTHING").
	// 	Exec(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	err = global.DB().NewSelect().Model(user).Where("phone = ?", phone).Scan(ctx)
	if err != nil {
		return nil, err
	}

	r.cache.KeyDelayDoubleDel(ctx, time.Millisecond*500, CacheKeyPrefixUser+user.ID.String())
	r.UpsertLatestIP(ctx, ids.UID(user.ID), ip)

	return user, nil
}

func (r *UserStore) DeleteByID(ctx context.Context, userId ids.UID) error {
	_, err := global.DB().NewDelete().Model((*models.User)(nil)).Where("id = ?", userId).Exec(ctx)
	if err == nil {
		key := CacheKeyPrefixUser + userId.String()
		ipkey := CacheKeyPrefixLatestIP + userId.String()
		r.cache.KeyDel(ctx, key, ipkey)
	}
	return err
}

func (r *UserStore) GetByID(ctx context.Context, userId ids.UID) (*models.User, error) {
	key := CacheKeyPrefixUser + userId.String()
	var user models.User
	err := r.cache.GetJson(ctx, key, &user)
	if err != nil {
		err := global.DB().NewSelect().Model(&user).Where("id = ?", userId).Scan(ctx)
		if err != nil {
			return nil, err
		}
		r.cache.SetJson(ctx, key, user, time.Hour)
		return &user, nil
	}
	return &user, nil
}

func (r *UserStore) UpsertLatestIP(ctx context.Context, userID ids.UID, ip string) error {
	ipInfo := &models.UserIP{
		ID:       userID,
		Register: ip,
		Latest:   ip,
	}
	_, err := global.DB().NewInsert().Model(ipInfo).
		On("CONFLICT (id) DO UPDATE").
		Set("latest = ?, updated_at = ?", ip, time.Now()).Exec(ctx)
	if err == nil {
		r.cache.KeyDelayDoubleDel(ctx, time.Millisecond*500, CacheKeyPrefixLatestIP+userID.String())
	}
	return err
}

func (r *UserStore) GetLatestIP(ctx context.Context, userId ids.UID) string {
	key := CacheKeyPrefixLatestIP + userId.String()
	ip, _ := r.cache.Get(ctx, key)
	if len(ip) != 0 {
		return ip
	}

	uipv := new(models.UserIP)
	err := global.DB().NewSelect().Model(uipv).Column("latest").Where("id = ?", userId).Scan(ctx)
	if err != nil {
		return ""
	}

	r.cache.Set(ctx, key, uipv.Latest, time.Hour)
	return uipv.Latest
}
