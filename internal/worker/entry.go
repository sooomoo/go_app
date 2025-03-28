package worker

import (
	"context"
	"fmt"
	"goapp/internal/worker/global"
	"goapp/internal/worker/tasks"
	"os"
	"os/signal"
	"syscall"
)

func Start() {
	// Initialize global instances
	ctx, cancel := context.WithCancel(context.Background())
	err := global.InitInstances(ctx, "", "", "")
	if err != nil {
		panic(err)
	}
	// Start tasks
	tasks.StartLogWriteTask(ctx)

	// Wait system signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	fmt.Printf("收到信号: %v\n", sig)
	cancel()
	// Cleanup Resources
	global.GetCache().Close()
	// TODO
	os.Exit(0)
}
