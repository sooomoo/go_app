package models

import (
	"context"
	"goapp/pkg/db"
	"goapp/pkg/ids"
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"users,alias:u"`
	ID            ids.UID   `bun:"id,pk" json:"id"`
	Phone         string    `bun:"phone,notnull" json:"phone"`
	Name          string    `bun:"name,notnull" json:"name"`
	Password      string    `bun:"password,notnull" json:"password"`
	Role          int32     `bun:"role,notnull" json:"role"`
	Profiles      db.JSON   `bun:"profiles" json:"profiles"`
	Invite        db.JSON   `bun:"invite" json:"invite"`
	Status        int16     `bun:"status,notnull" json:"status"`
	CreatedAt     int64     `bun:"created_at,notnull" json:"createdAt"`
	UpdatedAt     int64     `bun:"updated_at,notnull" json:"updatedAt"`
	DeletedAt     time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deletedAt"`
}

var _ bun.BeforeAppendModelHook = (*User)(nil)

func (u *User) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		u.CreatedAt = time.Now().Unix()
		u.UpdatedAt = time.Now().Unix()
	case *bun.UpdateQuery:
		u.UpdatedAt = time.Now().Unix()
	}
	return nil
}
