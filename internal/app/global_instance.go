package app

import (
	"context"
	"goapp/internal/app/hubs"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/sooomo/niu"
)

var pool niu.RoutinePool

func GetPool() niu.RoutinePool { return pool }

var cache *niu.Cache

func GetCache() *niu.Cache { return cache }

var idGen *niu.DistributeId

func GetIdGenerator() *niu.DistributeId { return idGen }

var locker *niu.DistributeLocker

func GetLocker() *niu.DistributeLocker { return locker }

var queue niu.MessageQueue

func GetQueue() niu.MessageQueue { return queue }

var snowflake *niu.Snowflake

func GetSnowflake() *niu.Snowflake { return snowflake }

func GetChatHub() *niu.Hub { return hubs.GetChatHub() }

func InitGlobalInstances(ctx context.Context, cacheAddr, lockerAddr, queueAddr string) error {
	var err error = nil
	pool, err = ants.NewPool(500000, ants.WithExpiryDuration(5*time.Minute))
	if err != nil {
		return err
	}

	cache, err = niu.NewCacheWithAddr(ctx, cacheAddr, "")
	if err != nil {
		return err
	}
	idGen, err = niu.NewDistributeId(ctx, cache.Master(), "idgen:user", 100000)
	if err != nil {
		return err
	}
	locker, err = niu.NewDistributeLockerWithAddr(ctx, lockerAddr, 15*time.Second, niu.LinearRetryStrategy(2*time.Second))
	if err != nil {
		return err
	}

	queue, err = niu.NewRedisMessageQueueWithAddr(ctx, queueAddr, pool, 1024, 100)
	if err != nil {
		return err
	}
	snowflake = niu.NewSnowflake(1)

	niu.InitSignHeaders("niu")
	err = hubs.StartChatHub(pool, []string{"niu-v1"})
	if err != nil {
		panic("hub start fail")
	}
	return err
}
