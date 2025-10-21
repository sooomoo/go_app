// 工作流管理器
package workflow

import (
	"context"
	"fmt"
	"goapp/pkg/core"
	"goapp/pkg/db"
	"goapp/pkg/ids"

	"github.com/uptrace/bun"
)

// Launch 启动工作流
func Launch(ctx context.Context, workflowID ids.UID, input core.MapX) error {
	var workflow WorkflowEntity
	err := db.Get().NewSelect().Model(&workflow).Where("id = ?", workflowID).Scan(ctx)
	if err != nil {
		return err
	}

	return workflow.run(ctx, input)
}

// Update 更新工作流
func Update(ctx context.Context, workflowID ids.UID, updates map[string]any) error {
	_, err := db.Update[WorkflowEntity](ctx, func(q *bun.UpdateQuery) {
		q.Where("id = ?", workflowID)
		for k, v := range updates {
			q.Set(fmt.Sprintf("%s = ?", k), v)
		}
	})
	return err
}

// List 列出工作流
func List(ctx context.Context, page, pageSize int) (*db.ListResult[WorkflowEntity], error) {
	var workflows []WorkflowEntity
	count, err := db.Get().NewSelect().Model(&workflows).Count(ctx)
	if err != nil {
		return nil, err
	}

	err = db.Get().NewSelect().Model(&workflows).Limit(pageSize).Offset((page - 1) * pageSize).Scan(ctx)
	return &db.ListResult[WorkflowEntity]{
		Total: count,
		Items: workflows,
	}, err
}

// ListSessions 列出工作流会话
func ListSessions(ctx context.Context, workflowID ids.UID, page, pageSize int) (*db.ListResult[WorkflowSessionEntity], error) {
	var sessions []WorkflowSessionEntity
	count, err := db.Get().NewSelect().Model(&sessions).Where("flow_id = ?", workflowID).Count(ctx)
	if err != nil {
		return nil, err
	}

	err = db.Get().NewSelect().Model(&sessions).Where("flow_id = ?", workflowID).Order("created_at desc").Limit(pageSize).Offset((page - 1) * pageSize).Scan(ctx)
	return &db.ListResult[WorkflowSessionEntity]{
		Total: count,
		Items: sessions,
	}, err
}

// ListSessionTasks 列出工作流会话任务
func ListSessionTasks(ctx context.Context, sessionID ids.UID) ([]WorkflowSessionTaskEntity, error) {
	var tasks []WorkflowSessionTaskEntity
	err := db.Get().NewSelect().Model(&tasks).Where("session_id = ?", sessionID).Scan(ctx)
	return tasks, err
}
