package tasks

import "context"

type Task interface {
	Run(ctx context.Context) error
}
