package workflow

import (
	"context"
	"errors"
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

type WorkflowKey string

var workflowIDKey WorkflowKey
var workflowSessionIDKey WorkflowKey
var workflowSessionTaskIDKey WorkflowKey

// WorkflowIDFromContext 从上下文中获取工作流的ID
func WorkflowIDFromContext(ctx context.Context) ids.UID {
	id, ok := ctx.Value(workflowIDKey).(ids.UID)
	if !ok {
		return ids.ZeroUID
	}
	return id
}

// WorkflowSessionIDFromContext 从上下文中获取工作流会话的ID
func WorkflowSessionIDFromContext(ctx context.Context) ids.UID {
	id, ok := ctx.Value(workflowSessionIDKey).(ids.UID)
	if !ok {
		return ids.ZeroUID
	}
	return id
}

// WorkflowSessionTaskIDFromContext 从上下文中获取工作流会话任务的ID
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
}

func (w *WorkflowEntity) startSession(ctx context.Context, input core.MapX) (*WorkflowSessionEntity, error) {
	session := &WorkflowSessionEntity{
		ID:         ids.NewUID(),
		FlowID:     w.ID,
		TaskNames:  w.TaskNames,
		Input:      db.JSON(input),
		Status:     WorkflowStatusRunning,
		StatusText: "running",
		BaseModelCreateUpdate: db.BaseModelCreateUpdate{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	_, err := db.Get().NewInsert().Model(session).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (w *WorkflowEntity) startTask(ctx context.Context, sessionID ids.UID, taskName string, input core.MapX) (*WorkflowSessionTaskEntity, error) {
	sessionTask := &WorkflowSessionTaskEntity{
		ID:         ids.NewUID(),
		TaskName:   taskName,
		FlowID:     w.ID,
		SessionID:  sessionID,
		Input:      db.JSON(input),
		Status:     WorkflowStatusRunning,
		StatusText: "running",
		BaseModelCreateUpdate: db.BaseModelCreateUpdate{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	_, err := db.Get().NewInsert().Model(sessionTask).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sessionTask, nil
}

// run 方法顺序执行工作流中的所有任务
func (w *WorkflowEntity) run(ctx context.Context, input core.MapX) (err error) {
	session, err := w.startSession(ctx, input)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, workflowIDKey, w.ID)
	ctx = context.WithValue(ctx, workflowSessionIDKey, session.ID)

	curFlowData := input
	for i, taskName := range w.TaskNames {
		if err = ctx.Err(); err != nil {
			break // 被取消了
		}
		err = session.stepForword(ctx, i, taskName)
		err = errors.Join(err, ctx.Err())
		if err != nil {
			break
		}
		fmt.Printf("Executing task %d(%s) in workflow '%s'\n", i+1, taskName, w.Name)
		taskEntity, er := w.startTask(ctx, session.ID, taskName, curFlowData)
		err = errors.Join(er, ctx.Err())
		if err != nil {
			break
		}
		curFlowData, err = taskEntity.execute(ctx)
		if err != nil {
			break
		}
	}

	return session.finish(ctx, curFlowData, err)
}

// 工作流运行历史记录
type WorkflowSessionEntity struct {
	db.BaseModelCreateUpdate `bun:"workflow_sessions,alias:wfs"`
	ID                       ids.UID        `bun:"id,pk" json:"id"`
	FlowID                   ids.UID        `bun:"flow_id,notnull" json:"flowId"`
	TaskNames                []string       `bun:"taskNames,notnull" json:"taskNames"` // 对当前工作流中的任务做个备份
	CurrentTask              string         `bun:"current_task" json:"currentTask"`    // 当前正在执行的任务
	Progress                 float64        `bun:"progress" json:"progress"`           // 当前工作流的进度，0-1
	Input                    db.JSON        `bun:"input" json:"input"`
	Output                   db.JSON        `bun:"output" json:"output"`
	Status                   WorkflowStatus `bun:"status" json:"status"`
	StatusText               string         `bun:"status_text" json:"statusText"`
}

func (w *WorkflowSessionEntity) stepForword(ctx context.Context, index int, taskName string) error {
	w.CurrentTask = taskName
	w.Progress = float64(index) / float64(len(w.TaskNames))
	_, err := db.Get().NewUpdate().Model(w).Where("id = ?", w.ID).
		Set("current_task = ?", taskName).
		Set("progress = ?", w.Progress).
		Set("updated_at = ?", time.Now()).
		Exec(ctx)
	return err
}

func (w *WorkflowSessionEntity) finish(ctx context.Context, output core.MapX, err error) error {
	u := db.Get().NewUpdate().Model(w).Where("id = ?", w.ID)
	if err != nil {
		w.Status = WorkflowStatusFail
		w.StatusText = err.Error()
	} else {
		w.Output = db.JSON(output)
		w.Status = WorkflowStatusSuccess
		w.StatusText = "success"
		u.Set("output = ?", output)
	}
	u.Set("status = ?", w.Status)
	u.Set("status_text = ?", w.StatusText)
	_, err = u.Exec(ctx)
	return err
}

type WorkflowSessionTaskEntity struct {
	db.BaseModelCreateUpdate `bun:"workflow_session_tasks,alias:wfst"`
	ID                       ids.UID        `bun:"id,pk" json:"id"`
	TaskName                 string         `bun:"task_name,notnull" json:"taskName"`
	FlowID                   ids.UID        `bun:"flow_id,notnull" json:"flowId"`
	SessionID                ids.UID        `bun:"session_id,notnull" json:"sessionId"`
	Input                    db.JSON        `bun:"input" json:"input"`
	Output                   db.JSON        `bun:"output" json:"output"`
	Status                   WorkflowStatus `bun:"status" json:"status"`
	StatusText               string         `bun:"status_text" json:"statusText"`
}

func (t *WorkflowSessionTaskEntity) execute(ctx context.Context) (core.MapX, error) {
	ctx = context.WithValue(ctx, workflowSessionTaskIDKey, t.ID)
	task := NewTask(t.TaskName)
	output, err := task.Execute(ctx, core.MapX(t.Input)) // execute task

	u := db.Get().NewUpdate().Model(t).Where("id = ?", t.ID)
	if err != nil {
		t.Status = WorkflowStatusFail
		t.StatusText = err.Error()
	} else {
		t.Output = db.JSON(output)
		t.Status = WorkflowStatusSuccess
		t.StatusText = "success"
		u.Set("output = ?", output)
	}
	u.Set("status = ?", t.Status)
	u.Set("status_text = ?", t.StatusText)
	_, err = db.Get().NewUpdate().Model(t).Where("id = ?", t.ID).Exec(ctx)

	return output, err
}
