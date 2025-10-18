package flows

// 任务实现
var taskImpls = make(map[string]func() Task)

// RegisterTask 注册任务实现: 应用启动时调用，后续不应该再调用
func RegisterTaskFactory(taskName string, factory func() Task) {
	taskImpls[taskName] = factory
}

// NewTask 获取任务实现, 调用factory生成实例
//
// 如果没有找到对应的factory，则 panic
func NewTask(taskName string) Task {
	factory, ok := taskImpls[taskName]
	if !ok {
		panic("task not found: " + taskName)
	}
	return factory()
}
