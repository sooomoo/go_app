package tasks

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

type TaskWebSearch struct {
	bun.BaseModel `bun:"task_web_searchs,alias:tws"`

	ID         string  `bun:"id,pk" json:"id"`
	Keywords   string  `bun:"keywords,notnull" json:"keywords"`
	TraceID    string  `bun:"trace_id,notnull" json:"traceId"`
	Progress   float32 `bun:"progress,notnull" json:"progress"`
	Status     int16   `bun:"status,notnull" json:"status"`
	StatusText string  `bun:"status_text,notnull" json:"statusText"`
	Result     string  `bun:"result" json:"result"`
	CreatedAt  int64   `bun:"created_at,notnull" json:"createdAt"`
	UpdatedAt  int64   `bun:"updated_at,notnull" json:"updatedAt"`
	SearchAt   int64   `bun:"search_at,notnull" json:"searchAt"`
	FinishAt   int64   `bun:"finish_at,notnull" json:"finishAt"`
}

var _ bun.BeforeAppendModelHook = (*TaskWebSearch)(nil)

func (u *TaskWebSearch) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		u.CreatedAt = time.Now().Unix()
		u.UpdatedAt = time.Now().Unix()
	case *bun.UpdateQuery:
		u.UpdatedAt = time.Now().Unix()
	}
	return nil
}
