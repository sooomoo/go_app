package flows

import (
	"context"
	"fmt"
	"goapp/pkg/core"
	"goapp/pkg/db"
	"goapp/pkg/ids"
	"time"
)

type WorkflowStatus string

const (
	WorkflowStatusUnknown = ""
	WorkflowStatusRunning = "running"
	WorkflowStatusSuccess = "success"
	WorkflowStatusFail    = "fail"
)

// Workflow 结构体表示一个包含多个任务的工作流
type WorkflowEntity struct {
	ID          ids.UID
	Name        string
	Description string
	TaskNames   []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   time.Time

	session *WorkflowSessionEntity
}

func (WorkflowEntity) TableName() string {
	return "workflows"
}

func (w *WorkflowEntity) instantialTasks() []Task {
	tasks := make([]Task, 0, len(w.TaskNames))
	for _, taskName := range w.TaskNames {
		tasks = append(tasks, NewTask(taskName))
	}
	return tasks
}

func (w *WorkflowEntity) startSession(ctx context.Context, fn func(input core.MapX) (core.MapX, error), input core.MapX) error {
	w.session = &WorkflowSessionEntity{
		ID:         ids.NewUID(),
		Input:      db.JSON(input),
		Status:     WorkflowStatusRunning,
		StatusText: "running",
		CreatedAt:  time.Now(),
	}
	_, err := db.Get().NewInsert().Model(w.session).Exec(ctx)
	if err != nil {
		return err
	}
	output, err := fn(input) // 执行工作流
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
		SessionID:  w.session.ID,
		Input:      db.JSON(input),
		Status:     WorkflowStatusRunning,
		StatusText: "running",
		CreatedAt:  time.Now(),
	}
	_, err := db.Get().NewInsert().Model(sessionTask).Exec(ctx)
	if err != nil {
		return nil, err
	}

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
	return w.startSession(ctx, func(input core.MapX) (core.MapX, error) {
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
	ID         ids.UID
	Input      db.JSON
	Output     db.JSON
	Status     WorkflowStatus
	StatusText string
	CreatedAt  time.Time
}

func (WorkflowSessionEntity) TableName() string {
	return "workflow_sessions"
}

type WorkflowSessionTaskEntity struct {
	ID         ids.UID
	SessionID  ids.UID
	Input      db.JSON
	Output     db.JSON
	Status     WorkflowStatus
	StatusText string
	CreatedAt  time.Time
}

func (WorkflowSessionTaskEntity) TableName() string {
	return "workflow_session_tasks"
}

type WorkflowLauncher struct {
}

func NewWorkflowLauncher() *WorkflowLauncher {
	return &WorkflowLauncher{}
}

func (l *WorkflowLauncher) Launch(ctx context.Context, workflowName string, input core.MapX) error {
	var workflow WorkflowEntity
	err := db.Get().NewSelect().Model(&workflow).Where("name = ?", workflowName).Scan(ctx)
	if err != nil {
		return err
	}

	return workflow.Run(ctx, input)
}
