// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

import "goapp/pkg/core"

const TableNameTaskWebSearch = "task_web_searches"

// TaskWebSearch 搜索任务
type TaskWebSearch struct {
	ID         int64        `gorm:"column:id;primaryKey" json:"id"`
	Keywords   string       `gorm:"column:keywords;not null;comment:搜索词，多个搜索词以逗号分隔" json:"keywords"` // 搜索词，多个搜索词以逗号分隔
	TraceID    string       `gorm:"column:trace_id;not null;comment:多个 ID 以逗号分隔" json:"traceId"`     // 多个 ID 以逗号分隔
	Status     uint8        `gorm:"column:status;not null" json:"status"`
	StatusText string       `gorm:"column:status_text;not null" json:"statusText"`
	Result     core.SqlJSON `gorm:"column:result" json:"result"`
	CreatedAt  int64        `gorm:"column:created_at;not null;comment:由 app 创建" json:"createdAt"`              // 由 app 创建
	SearchAt   int64        `gorm:"column:search_at;not null;comment:什么时候开始的搜索：由 searcher 更新" json:"searchAt"` // 什么时候开始的搜索：由 searcher 更新
	FinishAt   int64        `gorm:"column:finish_at;not null;comment:什么时候结束的搜索：由 searcher 更新" json:"finishAt"` // 什么时候结束的搜索：由 searcher 更新
}

// TableName TaskWebSearch's table name
func (*TaskWebSearch) TableName() string {
	return TableNameTaskWebSearch
}
