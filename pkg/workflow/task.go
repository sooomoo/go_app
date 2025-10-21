package workflow

import (
	"context"
	"goapp/pkg/core"
	"maps"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Task represents a task that can be executed.
type Task interface {
	// Name 返回任务的名称：固定值
	Name() string
	// Execute 执行任务
	//
	// ctx 表示上下文
	//
	// input 表示任务的输入
	//
	// 返回值: (output core.MapX, err error)
	Execute(ctx context.Context, input core.MapX) (core.MapX, error)
}

type RetryDelayFunc func(retry int) time.Duration
type RetryRetryableFunc func(err error) bool

// RetryableTask 可重试的任务
type RetryableTask struct {
	Task
	MaxRetries int
	RetryDelay RetryDelayFunc
	Retryable  RetryRetryableFunc
}

func NewRetryableTask(task Task, maxRetries int, retryDelay RetryDelayFunc, retryable RetryRetryableFunc) *RetryableTask {
	return &RetryableTask{
		Task:       task,
		MaxRetries: maxRetries,
		RetryDelay: retryDelay,
		Retryable:  retryable,
	}
}

func (RetryableTask) Name() string {
	return "retryable_task"
}

func (t RetryableTask) Execute(ctx context.Context, input core.MapX) (core.MapX, error) {
	output, err := t.Task.Execute(ctx, input)
	if err != nil && t.Retryable(err) {
		for i := range t.MaxRetries {
			time.Sleep(t.RetryDelay(i))
			output, err = t.Task.Execute(ctx, input)
			if err == nil {
				return output, nil
			}
		}
	}
	return output, err
}

// ParallelTask 需要多个任务同时执行并输出结果的情况
type ParallelTask struct {
	Tasks []Task
}

func (ParallelTask) Name() string {
	return "parallel_task"
}

func (t ParallelTask) Execute(ctx context.Context, input core.MapX) (core.MapX, error) {
	g, ctx := errgroup.WithContext(ctx)
	type tmpOutput struct {
		name   string
		output core.MapX
	}
	outputChan := make(chan *tmpOutput, len(t.Tasks))
	defer close(outputChan)

	for _, v := range t.Tasks {
		task := v // 为闭包创建局部变量
		g.Go(func() error {
			output, err := task.Execute(ctx, input)
			if err != nil {
				return err
			}
			outputChan <- &tmpOutput{name: task.Name(), output: output}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// 合并结果
	output := core.MapX{}
	for taskOutput := range outputChan {
		if len(taskOutput.output) == 0 {
			continue
		}
		maps.Copy(output, taskOutput.output)
		output[taskOutput.name] = output
	}
	return output, nil
}

type taskFactory struct {
	factories map[string]func() Task
	mutex     sync.RWMutex
}

func newTaskFactory() *taskFactory {
	return &taskFactory{
		factories: make(map[string]func() Task),
		mutex:     sync.RWMutex{},
	}
}

func (tf *taskFactory) RegisterTask(taskName string, factory func() Task) {
	tf.mutex.Lock()
	defer tf.mutex.Unlock()
	tf.factories[taskName] = factory
}

func (tf *taskFactory) NewTask(taskName string) Task {
	tf.mutex.RLock()
	defer tf.mutex.RUnlock()
	factory, ok := tf.factories[taskName]
	if !ok {
		panic("task not found: " + taskName)
	}
	return factory()
}

// 任务实现
var taskFactoryObj = newTaskFactory()

// RegisterTask 注册任务实现: 应用启动时调用，后续不应该再调用
func RegisterTaskFactory(taskName string, factory func() Task) {
	taskFactoryObj.RegisterTask(taskName, factory)
}

// NewTask 获取任务实现, 调用factory生成实例
//
// 如果没有找到对应的factory，则 panic
func NewTask(taskName string) Task {
	return taskFactoryObj.NewTask(taskName)
}
