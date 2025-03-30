package app

// 全局实例，用于全局使用，如缓存，消息队列，分布式锁，分布式ID生成器，分布式认证器，分布式消息总线等
import (
	"context"
	"goapp/internal/app/global"
	"goapp/internal/app/handlers/hubs"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sooomo/niu"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Init(ctx context.Context, configFile string) error {
	// load config first
	// TODO

	appConfig := global.AppConfig

	var err error = nil
	global.Pool, err = ants.NewPool(500000, ants.WithExpiryDuration(5*time.Minute))
	if err != nil {
		return err
	}

	global.Db, err = gorm.Open(mysql.Open(global.AppConfig.Db.ConnectString))
	if err != nil {
		return err
	}

	global.Cache, err = niu.NewCacheWithAddr(ctx, appConfig.Cache.Addr, appConfig.Cache.SlaveAddr)
	if err != nil {
		return err
	}

	global.DistributeId, err = niu.NewDistributeId(ctx, &redis.Options{
		Addr: appConfig.Cache.Addr,
	})
	if err != nil {
		return err
	}

	global.Locker, err = niu.NewDistributeLocker(
		ctx, &redis.Options{Addr: appConfig.Locker.Addr},
		time.Duration(appConfig.Locker.Ttl)*time.Second,
		niu.LinearRetryStrategy(time.Duration(appConfig.Locker.Backoff)*time.Second))
	if err != nil {
		return err
	}

	global.Queue, err = niu.NewRedisMessageQueue(ctx, &redis.Options{Addr: appConfig.Queue.Addr}, global.Pool, appConfig.Queue.XAddMaxLen, appConfig.Queue.BatchSize)
	if err != nil {
		return err
	}

	global.Snowflake = niu.NewSnowflake(appConfig.WorkerId)

	global.ChatHub, err = hubs.StartChatHub(global.Pool, &appConfig.Hub)
	if err != nil {
		panic("hub start fail")
	}
	return err
}

func Release() {
	if sqlDB, err := global.Db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	global.Pool.Release()
	global.Cache.Close()
	global.ChatHub.Close(10 * time.Second)
	global.DistributeId.Close()
	global.Locker.Close()
	global.Queue.Close()
}
