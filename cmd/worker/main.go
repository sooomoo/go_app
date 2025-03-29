package main

import (
	"context"
	"goapp/internal/worker/global"
	"goapp/internal/worker/tasks"

	"github.com/sooomo/niu"
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
	niu.WaitSysSignal(func() {
		cancel()
		// Cleanup Resources
		global.GetCache().Close()
	})
}
