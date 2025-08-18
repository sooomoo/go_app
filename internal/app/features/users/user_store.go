package users

import (
	"context"
	"goapp/internal/app/global"
	"goapp/internal/app/models"
	"goapp/pkg/cache"
	"goapp/pkg/ids"
	"time"
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

func (r *UserStore) Upsert(ctx context.Context, phone, ip string) (*models.User, error) {
	user := &models.User{
		ID:     ids.NewUID(),
		Phone:  phone,
		Name:   phone[3:6] + "****" + phone[10:],
		Role:   int32(RoleNormal),
		Status: UserStatusNormal,
	}

	_, err := global.DB().NewInsert().Model(user).
		On("CONFLICT (phone) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	err = global.DB().NewSelect().Model(user).Where("phone = ?", phone).Scan(ctx)
	if err != nil {
		return nil, err
	}

	r.cache.SetJson(ctx, CacheKeyPrefixUser+user.ID.String(), user, time.Hour)
	r.UpsertLatestIP(ctx, ids.UID(user.ID), ip)

	return user, nil
}

func (r *UserStore) GetById(ctx context.Context, userId ids.UID) (*models.User, error) {
	key := CacheKeyPrefixUser + userId.String()
	var user models.User
	err := r.cache.GetJson(ctx, key, &user)
	if err != nil {
		user := new(models.User)
		err := global.DB().NewSelect().Model(user).Where("id = ?", userId).Scan(ctx)
		if err != nil {
			return nil, err
		}
		r.cache.SetJson(ctx, key, user, time.Hour)
		return user, nil
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
		Set("latest = ?, updated_at = ?", ip, time.Now().Unix()).Exec(ctx)
	if err == nil {
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

	uipv := new(models.UserIP)
	err := global.DB().NewSelect().Model(uipv).Where("id = ?", userId).Scan(ctx)
	if err != nil {
		return ""
	}

	r.cache.Set(ctx, key, uipv.Latest, time.Hour)
	return uipv.Latest
}
