package models

import (
	"context"
	"goapp/pkg/db"
	"goapp/pkg/ids"
	"time"

	"github.com/uptrace/bun"
)

type UserStatus int16

const (
	UserStatusNormal UserStatus = 0
	UserStatusBanned UserStatus = 1 // 已封禁
)

type UserRole int32

const (
	UserRoleNormal UserRole = 0b00000000 // 普通用户
	UserRolePro    UserRole = 0b00000001 // Pro用户
	UserRoleAdmin  UserRole = 0b10000000 // 管理员
)

type User struct {
	bun.BaseModel `bun:"users,alias:u"`
	ID            ids.UID    `bun:"id,pk" json:"id"`
	Phone         string     `bun:"phone,notnull" json:"phone"`
	Name          string     `bun:"name,notnull" json:"name"`
	Password      string     `bun:"password,notnull" json:"password"`
	Role          UserRole   `bun:"role,notnull" json:"role"`
	Profiles      db.JSON    `bun:"profiles" json:"profiles"`
	Invite        db.JSON    `bun:"invite" json:"invite"`
	Status        UserStatus `bun:"status,notnull" json:"status"`
	CreatedAt     time.Time  `bun:"created_at,notnull" json:"createdAt"`
	UpdatedAt     time.Time  `bun:"updated_at,notnull" json:"updatedAt"`
	DeletedAt     time.Time  `bun:"deleted_at,soft_delete,nullzero" json:"deletedAt"`
}

var _ bun.BeforeAppendModelHook = (*User)(nil)

func (u *User) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		u.CreatedAt = time.Now()
		u.UpdatedAt = time.Now()
	case *bun.UpdateQuery:
		u.UpdatedAt = time.Now()
	}
	return nil
}
