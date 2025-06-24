package tasks

import (
	"context"
	"fmt"
	"goapp/internal/workers/worker/global"
)

func StartLogWriteTask(ctx context.Context) {
	global.GetQueue().Subscribe(ctx, "log", "log_writer", global.GetAppId(), writeLog)
}

func writeLog(ctx context.Context, id string, msg map[string]any) error {
	fmt.Printf("收到消息: %v\n", msg)
	return nil
}
