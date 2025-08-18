package models

import (
	"context"
	"goapp/pkg/ids"
	"time"

	"github.com/uptrace/bun"
)

type UserAccountBinding struct {
	bun.BaseModel `bun:"table:user_account_bindings,alias:ua"`
	ID            ids.UID `bun:"id,pk" json:"id"`
	AccountType   int32   `bun:"account_type" json:"accountType"`
	Account       string  `bun:"account" json:"account"`
	Status        int16   `bun:"status" json:"status"`
	StatusText    string  `bun:"status_text" json:"statusText"`
	UserID        ids.UID `bun:"user_id" json:"userId"` // 用户 ID
	CreatedAt     int64   `bun:"created_at" json:"createdAt"`
	UpdatedAt     int64   `bun:"updated_at" json:"updatedAt"`
	DeletedAt     *int64  `bun:"deleted_at,soft_delete" json:"deletedAt"`
}

var _ bun.BeforeAppendModelHook = (*UserAccountBinding)(nil)

func (u *UserAccountBinding) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		u.CreatedAt = time.Now().Unix()
		u.UpdatedAt = time.Now().Unix()
	case *bun.UpdateQuery:
		u.UpdatedAt = time.Now().Unix()
	}
	return nil
}
