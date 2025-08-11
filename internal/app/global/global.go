package global

import (
	"context"
	"fmt"
	"goapp/internal/app/dao/query"
	"goapp/pkg/cache"
	"goapp/pkg/core"
	"goapp/pkg/distribute"
	"goapp/pkg/ids"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var mut sync.RWMutex
var inited atomic.Bool
var released atomic.Bool

var pool core.CoroutinePool
var cach *cache.Cache
var locker *distribute.Locker
var queue distribute.MessageQueue
var appConfig *AppConfig
var db *gorm.DB

func Init(ctx context.Context) {
	mut.Lock()
	defer mut.Unlock()

	if inited.Load() {
		return
	}
	defer inited.Store(true)

	// load config first
	err := loadConfig()
	if err != nil {
		panic(err)
	}
	appConfig.Id = ids.NewUUID()

	pool, err = ants.NewPool(500000, ants.WithExpiryDuration(5*time.Minute))
	if err != nil {
		panic(err)
	}

	dbMaster := appConfig.Database.ConnectString
	dbSlaves := []string{appConfig.Database.ConnectString}
	db, err = core.InitReplicasDB(dbMaster, dbSlaves, 10*time.Second)
	if err != nil {
		panic(err)
	}
	// 设置默认的 Db 连接
	query.SetDefault(db)

	cach, err = cache.NewCache(ctx, appConfig.Cache.GetRedisOption(), nil)
	if err != nil {
		panic(err)
	}

	locker, err = distribute.NewLocker(
		ctx, appConfig.Locker.GetRedisOption(),
		distribute.WithDefaultTtl(time.Duration(appConfig.Locker.Ttl)*time.Second),
		distribute.WithDefaultRetryStrategy(distribute.LinearRetryStrategy(time.Duration(appConfig.Locker.Backoff)*time.Second)))
	if err != nil {
		panic(err)
	}

	queue, err = distribute.NewRedisMessageQueue(ctx, appConfig.Queue.GetRedisOption(), pool, appConfig.Queue.XAddMaxLen, appConfig.Queue.BatchSize)
	if err != nil {
		panic(err)
	}
}

func loadConfig() error {
	env := os.Getenv("env")
	// 设置配置文件名称和类型
	viper.SetConfigName(fmt.Sprintf("config.%s", env))
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatal().Msgf("配置文件未找到: %v", err)
		} else {
			log.Fatal().Msgf("读取配置文件出错: %v", err)
		}
		return err
	}

	// 解析配置到结构体
	var config AppConfig
	err := viper.Unmarshal(&config)
	if err != nil {
		log.Fatal().Msgf("无法解析配置文件: %v", err)
		return err
	}

	appConfig = &config

	// 打印初始配置
	log.Info().Any("配置如下", config).Msg("配置加载完成。。。")
	return nil
}

func Release() {
	mut.Lock()
	defer mut.Unlock()

	if released.Load() {
		return
	}
	defer released.Store(true)

	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	pool.Release()
	cach.Close()
	locker.Close()
	queue.Close()
}

func GetDB() *gorm.DB {
	return db
}
func GetCoroutinePool() core.CoroutinePool {
	return pool
}
func GetCache() *cache.Cache {
	return cach
}

func GetLocker() *distribute.Locker {
	return locker
}
func GetQueue() distribute.MessageQueue {
	return queue
}
func GetAppConfig() *AppConfig {
	return appConfig
}
func GetAuthConfig() AuthenticatorConfig {
	return appConfig.Authenticator
}
