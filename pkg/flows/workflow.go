package flows

import (
	"context"
	"fmt"
	"goapp/pkg/core"
	"goapp/pkg/db"
	"goapp/pkg/ids"
	"time"

	"github.com/uptrace/bun"
)

type WorkflowStatus string

const (
	WorkflowStatusUnknown = ""
	WorkflowStatusRunning = "running"
	WorkflowStatusSuccess = "success"
	WorkflowStatusFail    = "fail"
)

type WorkflowKey string

var workflowIDKey WorkflowKey
var workflowSessionIDKey WorkflowKey
var workflowSessionTaskIDKey WorkflowKey

func WorkflowIDFromContext(ctx context.Context) ids.UID {
	id, ok := ctx.Value(workflowIDKey).(ids.UID)
	if !ok {
		return ids.ZeroUID
	}
	return id
}

func WorkflowSessionIDFromContext(ctx context.Context) ids.UID {
	id, ok := ctx.Value(workflowSessionIDKey).(ids.UID)
	if !ok {
		return ids.ZeroUID
	}
	return id
}

func WorkflowSessionTaskIDFromContext(ctx context.Context) ids.UID {
	id, ok := ctx.Value(workflowSessionTaskIDKey).(ids.UID)
	if !ok {
		return ids.ZeroUID
	}
	return id
}

// Workflow 结构体表示一个包含多个任务的工作流
type WorkflowEntity struct {
	db.BaseModelCreateUpdateDelete `bun:"workflows,alias:wf"`
	ID                             ids.UID  `bun:"id,pk" json:"id"`
	Name                           string   `bun:"name,notnull" json:"name"`
	Description                    string   `bun:"description,notnull" json:"description"`
	TaskNames                      []string `bun:"taskNames,notnull" json:"taskNames"`

	session *WorkflowSessionEntity `bun:"-" json:"-"`
}

func (w *WorkflowEntity) instantialTasks() []Task {
	tasks := make([]Task, 0, len(w.TaskNames))
	for _, taskName := range w.TaskNames {
		tasks = append(tasks, NewTask(taskName))
	}
	return tasks
}

func (w *WorkflowEntity) startSession(ctx context.Context, fn func(ctx context.Context, input core.MapX) (core.MapX, error), input core.MapX) error {
	w.session = &WorkflowSessionEntity{
		ID:         ids.NewUID(),
		FlowID:     w.ID,
		Input:      db.JSON(input),
		Status:     WorkflowStatusRunning,
		StatusText: "running",
		BaseModelCreate: db.BaseModelCreate{
			CreatedAt: time.Now(),
		},
	}
	_, err := db.Get().NewInsert().Model(w.session).Exec(ctx)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, workflowIDKey, w.ID)
	ctx = context.WithValue(ctx, workflowSessionIDKey, w.session.ID)

	output, err := fn(ctx, input) // 执行工作流
	if err != nil {
		w.session.Status = WorkflowStatusFail
		w.session.StatusText = err.Error()
	} else {
		w.session.Output = db.JSON(output)
		w.session.Status = WorkflowStatusSuccess
		w.session.StatusText = "success"
	}
	if err != nil {
		return err
	}
	_, err = db.Get().NewUpdate().Model(w.session).Where("id = ?", w.session.ID).Exec(ctx)
	return err
}

func (w *WorkflowEntity) executeTask(ctx context.Context, task Task, input core.MapX) (core.MapX, error) {
	sessionTask := WorkflowSessionTaskEntity{
		ID:         ids.NewUID(),
		FlowID:     w.ID,
		SessionID:  w.session.ID,
		Input:      db.JSON(input),
		Status:     WorkflowStatusRunning,
		StatusText: "running",
		BaseModelCreate: db.BaseModelCreate{
			CreatedAt: time.Now(),
		},
	}
	_, err := db.Get().NewInsert().Model(sessionTask).Exec(ctx)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, workflowSessionTaskIDKey, sessionTask.ID)
	output, err := task.Execute(ctx, input) // 执行任务
	if err != nil {
		sessionTask.Status = WorkflowStatusFail
		sessionTask.StatusText = err.Error()
	} else {
		sessionTask.Output = db.JSON(output)
		sessionTask.Status = WorkflowStatusSuccess
		sessionTask.StatusText = "success"
	}
	if err != nil {
		return nil, err
	}
	_, err = db.Get().NewUpdate().Model(sessionTask).Where("id = ?", sessionTask.ID).Exec(ctx)
	return output, err
}

// Run 方法顺序执行工作流中的所有任务
func (w *WorkflowEntity) Run(ctx context.Context, input core.MapX) (err error) {
	return w.startSession(ctx, func(ctx context.Context, input core.MapX) (core.MapX, error) {
		tasks := w.instantialTasks()
		curFlowData := input
		for i, task := range tasks {
			// 检查协程是否取消方式一
			// select {
			// case <-ctx.Done():
			// 	return nil, ctx.Err()
			// default:
			// 	fmt.Printf("Executing task %d in workflow '%s'\n", i+1, w.Name)
			// 	curFlowData, err = w.executeTask(ctx, task, curFlowData)
			// 	if err != nil {
			// 		return nil, err
			// 	}
			// }

			// 检查协程是否取消方式二
			if err := ctx.Err(); err != nil {
				return nil, err // 被取消了
			}
			fmt.Printf("Executing task %d in workflow '%s'\n", i+1, w.Name)
			curFlowData, err = w.executeTask(ctx, task, curFlowData)
			if err != nil {
				return nil, err
			}
		}
		return curFlowData, nil
	}, input)
}

// 工作流运行历史记录
type WorkflowSessionEntity struct {
	db.BaseModelCreate `bun:"workflow_sessions,alias:wfs"`
	ID                 ids.UID        `bun:"id,pk" json:"id"`
	FlowID             ids.UID        `bun:"flow_id,notnull" json:"flowId"`
	Input              db.JSON        `bun:"input" json:"input"`
	Output             db.JSON        `bun:"output" json:"output"`
	Status             WorkflowStatus `bun:"status" json:"status"`
	StatusText         string         `bun:"status_text" json:"statusText"`
}

type WorkflowSessionTaskEntity struct {
	db.BaseModelCreate `bun:"workflow_session_tasks,alias:wfst"`
	ID                 ids.UID
	FlowID             ids.UID
	SessionID          ids.UID
	Input              db.JSON
	Output             db.JSON
	Status             WorkflowStatus
	StatusText         string
}

// 工作流管理器
type WorkflowManager struct {
}

func NewWorkflowManager() *WorkflowManager {
	return &WorkflowManager{}
}

func (WorkflowManager) Launch(ctx context.Context, workflowID ids.UID, input core.MapX) error {
	var workflow WorkflowEntity
	err := db.Get().NewSelect().Model(&workflow).Where("id = ?", workflowID).Scan(ctx)
	if err != nil {
		return err
	}

	return workflow.Run(ctx, input)
}

func (WorkflowManager) Update(ctx context.Context, workflowID ids.UID, updates map[string]any) error {
	_, err := db.Update[WorkflowEntity](ctx, func(q *bun.UpdateQuery) {
		q.Where("id = ?", workflowID)
		for k, v := range updates {
			q.Set(fmt.Sprintf("%s = ?", k), v)
		}
	})
	return err
}

type WorkflowManagerListResponse struct {
	Total int              `json:"total"`
	Items []WorkflowEntity `json:"items"`
}

func (WorkflowManager) List(ctx context.Context, page, pageSize int) (*WorkflowManagerListResponse, error) {
	var workflows []WorkflowEntity
	count, err := db.Get().NewSelect().Model(&workflows).Count(ctx)
	if err != nil {
		return nil, err
	}

	err = db.Get().NewSelect().Model(&workflows).Limit(pageSize).Offset((page - 1) * pageSize).Scan(ctx)
	return &WorkflowManagerListResponse{
		Total: count,
		Items: workflows,
	}, err
}
