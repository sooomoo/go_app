package app

// 全局实例，用于全局使用，如缓存，消息队列，分布式锁，分布式ID生成器，分布式认证器，分布式消息总线等
import (
	"context"
	"goapp/internal/app/config"
	"goapp/internal/app/global"
	"goapp/internal/app/routes/hubs"
	"goapp/pkg/cache"
	"goapp/pkg/core"
	"goapp/pkg/distribute"
	"time"

	"github.com/panjf2000/ants/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Init(ctx context.Context) error {
	var err error = nil
	// load config first
	global.AppConfig, err = config.Load()
	if err != nil {
		return err
	}
	global.AppConfig.Id = core.NewUUIDWithoutDash()

	appConfig := global.AppConfig

	global.Pool, err = ants.NewPool(500000, ants.WithExpiryDuration(5*time.Minute))
	if err != nil {
		return err
	}

	global.Db, err = gorm.Open(mysql.Open(global.AppConfig.Database.ConnectString))
	if err != nil {
		return err
	}

	global.Cache, err = cache.NewCache(ctx, appConfig.Cache.GetRedisOption(), nil)
	if err != nil {
		return err
	}

	global.DistributeId, err = distribute.NewDistributeId(ctx, appConfig.Cache.GetRedisOption())
	if err != nil {
		return err
	}

	global.UserIdGenerator, err = global.DistributeId.NewGenerator(ctx, "id_gen:user", 100000)
	if err != nil {
		return err
	}

	global.Locker, err = distribute.NewDistributeLocker(
		ctx, appConfig.Locker.GetRedisOption(),
		time.Duration(appConfig.Locker.Ttl)*time.Second,
		distribute.LinearRetryStrategy(time.Duration(appConfig.Locker.Backoff)*time.Second))
	if err != nil {
		return err
	}

	global.Queue, err = distribute.NewRedisMessageQueue(ctx, appConfig.Queue.GetRedisOption(), global.Pool, appConfig.Queue.XAddMaxLen, appConfig.Queue.BatchSize)
	if err != nil {
		return err
	}

	global.Snowflake = distribute.NewSnowflake(appConfig.WorkerId)

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
