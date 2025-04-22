package main

import (
	"context"
	"goapp/internal/worker/global"
	"goapp/internal/worker/tasks"
	"goapp/pkg/core"
)

func main() {
	// Initialize global instances
	ctx, cancel := context.WithCancel(context.Background())
	err := global.Init(ctx)
	if err != nil {
		panic(err)
	}

	// Start tasks
	tasks.StartLogWriteTask(ctx)

	// Wait system signal, and cleanup resources
	core.WaitSysSignal(func() {
		cancel()
		// Cleanup Resources
		global.Release()
	})
}
