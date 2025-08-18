package models

import (
	"context"
	"goapp/pkg/ids"
	"time"

	"github.com/uptrace/bun"
)

type UserIP struct {
	bun.BaseModel `bun:"user_ips,alias:uip"`
	ID            ids.UID `bun:"id,pk" json:"id"`
	Register      string  `bun:"register,notnull" json:"register"`
	Latest        string  `bun:"latest,notnull" json:"latest"`
	CreatedAt     int64   `bun:"created_at" json:"createdAt"`
	UpdatedAt     int64   `bun:"updated_at" json:"updatedAt"`
	DeletedAt     *int64  `bun:"deleted_at,soft_delete" json:"deletedAt"`
}

var _ bun.BeforeAppendModelHook = (*UserIP)(nil)

func (u *UserIP) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		u.CreatedAt = time.Now().Unix()
		u.UpdatedAt = time.Now().Unix()
	case *bun.UpdateQuery:
		u.UpdatedAt = time.Now().Unix()
	}
	return nil
}
