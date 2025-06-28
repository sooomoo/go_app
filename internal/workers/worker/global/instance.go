package global

import (
	"context"
	"goapp/pkg/cache"
	"goapp/pkg/core"
	"goapp/pkg/distribute"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/redis/go-redis/v9"
)

var appId string

func GetAppId() string { return appId }

var pool core.CoroutinePool

func GetPool() core.CoroutinePool { return pool }

var cacheInst *cache.Cache

func GetCache() *cache.Cache { return cacheInst }

var locker *distribute.Locker

func GetLocker() *distribute.Locker { return locker }

var queue distribute.MessageQueue

func GetQueue() distribute.MessageQueue { return queue }

func Init(ctx context.Context) error {
	appId = core.NewSeqID().Hex()

	var err error = nil
	pool, err = ants.NewPool(100000, ants.WithExpiryDuration(5*time.Minute))
	if err != nil {
		return err
	}

	cacheInst, err = cache.NewCacheWithAddr(ctx, "", "")
	if err != nil {
		return err
	}
	locker, err = distribute.NewLocker(ctx, &redis.Options{
		Addr: "",
	}, 15*time.Second, distribute.LinearRetryStrategy(2*time.Second))
	if err != nil {
		return err
	}

	queue, err = distribute.NewRedisMessageQueue(ctx, &redis.Options{
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
