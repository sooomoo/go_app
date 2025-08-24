package tasks

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

type TaskStatus int16

const (
	TaskStatusPending   TaskStatus = 0 // 待处理
	TaskStatusRunning   TaskStatus = 1 // 处理中
	TaskStatusCompleted TaskStatus = 2 // 已完成
	TaskStatusFailed    TaskStatus = 3 // 失败
)

type TaskWebSearch struct {
	bun.BaseModel `bun:"task_web_searchs,alias:tws"`

	ID         string     `bun:"id,pk" json:"id"`
	Keywords   string     `bun:"keywords,notnull" json:"keywords"`
	TraceID    string     `bun:"trace_id,notnull" json:"traceId"`
	Progress   float32    `bun:"progress,notnull" json:"progress"`
	Status     TaskStatus `bun:"status,notnull" json:"status"`
	StatusText string     `bun:"status_text,notnull" json:"statusText"`
	Result     string     `bun:"result" json:"result"`
	CreatedAt  time.Time  `bun:"created_at,notnull" json:"createdAt"`
	UpdatedAt  time.Time  `bun:"updated_at,notnull" json:"updatedAt"`
	SearchAt   *time.Time `bun:"search_at,notnull" json:"searchAt"`
	FinishAt   *time.Time `bun:"finish_at,notnull" json:"finishAt"`
}

var _ bun.BeforeAppendModelHook = (*TaskWebSearch)(nil)

func (u *TaskWebSearch) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		u.CreatedAt = time.Now()
		u.UpdatedAt = time.Now()
	case *bun.UpdateQuery:
		u.UpdatedAt = time.Now()
	}
	return nil
}
