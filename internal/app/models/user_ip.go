package models

import (
	"context"
	"goapp/pkg/ids"
	"time"

	"github.com/uptrace/bun"
)

type UserIP struct {
	bun.BaseModel `bun:"user_ips,alias:uip"`
	ID            ids.UID    `bun:"id,pk" json:"id"`
	Register      string     `bun:"register,notnull" json:"register"`
	Latest        string     `bun:"latest,notnull" json:"latest"`
	CreatedAt     time.Time  `bun:"created_at" json:"createdAt"`
	UpdatedAt     time.Time  `bun:"updated_at" json:"updatedAt"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete" json:"deletedAt"`
}

var _ bun.BeforeAppendModelHook = (*UserIP)(nil)

func (u *UserIP) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		u.CreatedAt = time.Now()
		u.UpdatedAt = time.Now()
	case *bun.UpdateQuery:
		u.UpdatedAt = time.Now()
	}
	return nil
}
