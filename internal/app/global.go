package app

import (
	"context"
	"fmt"
	"goapp/internal/app/stores/dao/query"
	"goapp/pkg/cache"
	"goapp/pkg/core"
	"goapp/pkg/distribute"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type GlobalInstance struct {
	mut      sync.RWMutex
	inited   atomic.Bool
	released atomic.Bool

	pool      core.CoroutinePool
	cache     *cache.Cache
	locker    *distribute.Locker
	queue     distribute.MessageQueue
	appConfig *AppConfig
	db        *gorm.DB
}

var (
	global *GlobalInstance = &GlobalInstance{}
)

// GetGlobal 获取全局实例
func GetGlobal() *GlobalInstance {
	return global
}

func (g *GlobalInstance) Init(ctx context.Context) {
	g.mut.Lock()
	defer g.mut.Unlock()

	if g.inited.Load() {
		return
	}
	defer g.inited.Store(true)

	// load config first
	err := g.loadConfig()
	if err != nil {
		panic(err)
	}
	g.appConfig.Id = core.NewSeqID().Hex()

	g.pool, err = ants.NewPool(500000, ants.WithExpiryDuration(5*time.Minute))
	if err != nil {
		panic(err)
	}

	g.db, err = core.InitDB(g.appConfig.Database.ConnectString, 10*time.Second)
	if err != nil {
		panic(err)
	}
	// 设置默认的 Db 连接
	query.SetDefault(g.db)

	g.cache, err = cache.NewCache(ctx, g.appConfig.Cache.GetRedisOption(), nil)
	if err != nil {
		panic(err)
	}

	g.locker, err = distribute.NewLocker(
		ctx, g.appConfig.Locker.GetRedisOption(),
		time.Duration(g.appConfig.Locker.Ttl)*time.Second,
		distribute.LinearRetryStrategy(time.Duration(g.appConfig.Locker.Backoff)*time.Second))
	if err != nil {
		panic(err)
	}

	g.queue, err = distribute.NewRedisMessageQueue(ctx, g.appConfig.Queue.GetRedisOption(), g.pool, g.appConfig.Queue.XAddMaxLen, g.appConfig.Queue.BatchSize)
	if err != nil {
		panic(err)
	}
}

func (g *GlobalInstance) loadConfig() error {
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

	g.appConfig = &config

	// 打印初始配置
	log.Info().Any("配置如下", config).Msg("配置加载完成。。。")
	return nil
}

func (g *GlobalInstance) Release() {
	g.mut.Lock()
	defer g.mut.Unlock()

	if g.released.Load() {
		return
	}
	defer g.released.Store(true)

	if sqlDB, err := g.db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	g.pool.Release()
	g.cache.Close()
	g.locker.Close()
	g.queue.Close()
}

func (g *GlobalInstance) GetDB() *gorm.DB {
	return g.db
}
func (g *GlobalInstance) GetCoroutinePool() core.CoroutinePool {
	return g.pool
}
func (g *GlobalInstance) GetCache() *cache.Cache {
	return g.cache
}

func (g *GlobalInstance) GetLocker() *distribute.Locker {
	return g.locker
}
func (g *GlobalInstance) GetQueue() distribute.MessageQueue {
	return g.queue
}
func (g *GlobalInstance) GetAppConfig() *AppConfig {
	return g.appConfig
}
func (g *GlobalInstance) GetAuthConfig() AuthenticatorConfig {
	return g.appConfig.Authenticator
}
