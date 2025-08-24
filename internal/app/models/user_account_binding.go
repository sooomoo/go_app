package models

import (
	"context"
	"goapp/pkg/ids"
	"time"

	"github.com/uptrace/bun"
)

type AccountType int32

const (
	AccountTypeEmail  AccountType = 1 // 邮箱
	AccountTypeWechat AccountType = 2 // Wechat
	AccountTypeQQ     AccountType = 3 // QQ
)

type AccountStatus int16

const (
	AccountStatusInactive AccountStatus = 0 // 未激活
	AccountStatusActive   AccountStatus = 1 // 已激活
	AccountStatusBanned   AccountStatus = 2 // 已封禁
)

type UserAccountBinding struct {
	bun.BaseModel `bun:"table:user_account_bindings,alias:ua"`

	ID          ids.UID       `bun:"id,pk" json:"id"`
	AccountType AccountType   `bun:"account_type" json:"accountType"`
	Account     string        `bun:"account" json:"account"`
	Status      AccountStatus `bun:"status" json:"status"`
	StatusText  string        `bun:"status_text" json:"statusText"`
	UserID      ids.UID       `bun:"user_id" json:"userId"` // 用户 ID
	CreatedAt   time.Time     `bun:"created_at" json:"createdAt"`
	UpdatedAt   time.Time     `bun:"updated_at" json:"updatedAt"`
	DeletedAt   *time.Time    `bun:"deleted_at,soft_delete" json:"deletedAt"`
}

var _ bun.BeforeAppendModelHook = (*UserAccountBinding)(nil)

func (u *UserAccountBinding) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		u.CreatedAt = time.Now()
		u.UpdatedAt = time.Now()
	case *bun.UpdateQuery:
		u.UpdatedAt = time.Now()
	}
	return nil
}
