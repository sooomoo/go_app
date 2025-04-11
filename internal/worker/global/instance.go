package global

import (
	"context"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sooomo/niu"
)

var appId string

func GetAppId() string { return appId }

var pool niu.CoroutinePool

func GetPool() niu.CoroutinePool { return pool }

var cache *niu.Cache

func GetCache() *niu.Cache { return cache }

var locker *niu.DistributeLocker

func GetLocker() *niu.DistributeLocker { return locker }

var queue niu.MessageQueue

func GetQueue() niu.MessageQueue { return queue }

func Init(ctx context.Context) error {
	appId = niu.NewUUIDWithoutDash()

	var err error = nil
	pool, err = ants.NewPool(100000, ants.WithExpiryDuration(5*time.Minute))
	if err != nil {
		return err
	}

	cache, err = niu.NewCacheWithAddr(ctx, "", "")
	if err != nil {
		return err
	}
	locker, err = niu.NewDistributeLocker(ctx, &redis.Options{
		Addr: "",
	}, 15*time.Second, niu.LinearRetryStrategy(2*time.Second))
	if err != nil {
		return err
	}

	queue, err = niu.NewRedisMessageQueue(ctx, &redis.Options{
		Addr: "",
	}, pool, 1024, 100)
	if err != nil {
		return err
	}

	return err
}

func Release() {
	if pool != nil {
		pool.Release()
	}
}
