package searcher

import (
	"context"
	"goapp/internal/workers/searcher/tasks"
	"goapp/pkg/rmq"
)

var client *rmq.Client

func Start(ctx context.Context) error {
	client = rmq.NewClient(GetGlobal().config.RMQ.Addr)
	err := client.Connect()
	if err != nil {
		return err
	}

	// start tasks
	kwTask := tasks.NewKeywordTask(client)
	err = kwTask.Run(ctx)
	if err != nil {
		return err
	}

	return nil
}
