package app

import (
	"context"
	"goapp/internal/app/repositories"
	"time"

	"github.com/google/wire"
	"github.com/panjf2000/ants/v2"
	"github.com/sooomo/niu"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewCoroutinePool() (niu.RoutinePool, error) {
	return ants.NewPool(500000, ants.WithExpiryDuration(30*time.Minute))
}

func NewCache(ctx context.Context, cfg *CacheConfig) (*niu.Cache, error) {
	return niu.NewCacheWithAddr(ctx, cfg.Addr, "")
}

func NewIdGenerator(ctx context.Context, cfg *CacheConfig) (*niu.DistributeId, error) {
	idGen, err := niu.NewDistributeIdWithAddr(ctx, cfg.Addr, "idgen:user", 100000)
	if err != nil {
		return nil, err
	}
	return idGen, nil
}

// 提供者返回错误
func NewDB(cfg *DBConfig) (*gorm.DB, func(), error) {
	db, err := gorm.Open(mysql.Open(cfg.ConnectString))
	if err != nil {
		return nil, nil, err
	}
	return db, func() {
		// 关闭数据库连接
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}, nil
}

var ProviderSet = wire.NewSet(
	NewCoroutinePool,
	NewCache,
	NewDB,
	repositories.NewRepositoryOfUser,
)

func Init() error {
	wire.Build(
		wire.Value(&AppConfig{}),
		NewCoroutinePool,
		NewCache,
		NewDB,
		// wire.Bind(new(Fooer), new(*MyFooer)),
		repositories.NewRepositoryOfUser,
	)
	return nil
}
