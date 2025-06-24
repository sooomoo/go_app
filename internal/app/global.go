package app

import (
	"context"
	"fmt"
	"goapp/internal/app/repositories/dao/query"
	"goapp/internal/pkg/services"
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
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

type GlobalInstance struct {
	mut      sync.RWMutex
	inited   atomic.Bool
	released atomic.Bool

	pool      core.CoroutinePool
	cache     *cache.Cache
	idService services.IDService
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
	g.appConfig.Id = core.NewUUIDWithoutDash()

	g.pool, err = ants.NewPool(500000, ants.WithExpiryDuration(5*time.Minute))
	if err != nil {
		panic(err)
	}

	g.db, err = gorm.Open(mysql.Open(g.appConfig.Database.ConnectString))
	if err != nil {
		panic(err)
	}
	// 连接池
	g.db.Use(
		dbresolver.Register(dbresolver.Config{
			Sources:  []gorm.Dialector{mysql.Open(g.appConfig.Database.ConnectString)},
			Replicas: []gorm.Dialector{mysql.Open(g.appConfig.Database.ConnectString), mysql.Open(g.appConfig.Database.ConnectString)},
			// sources/replicas load balancing policy
			Policy: dbresolver.RandomPolicy{},
			// print sources/replicas mode in logger
			TraceResolverMode: true,
		}).
			SetConnMaxIdleTime(time.Hour).
			SetConnMaxLifetime(2 * time.Hour).
			SetMaxIdleConns(100).
			SetMaxOpenConns(200),
	)
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

	g.idService = services.NewDefaultIDService(1)
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
func (g *GlobalInstance) GetIDService() services.IDService {
	return g.idService
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
