package core

import (
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

type AutoReconnectDialector struct {
	mysql.Dialector
	healthCheckInterval time.Duration
}

func (dialector AutoReconnectDialector) Initialize(db *gorm.DB) error {
	err := dialector.Dialector.Initialize(db)
	if err != nil {
		return err
	}

	// 启动健康检查
	go dialector.checkConnection(db)

	return nil
}

func (dialector AutoReconnectDialector) checkConnection(db *gorm.DB) {
	ticker := time.NewTicker(dialector.healthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		sqlDB, err := db.DB()
		if err != nil {
			_ = db.Config.Dialector.Initialize(db)
			continue
		}

		if err := sqlDB.Ping(); err != nil {
			_ = db.Config.Dialector.Initialize(db)
		}
	}
}

// 使用自定义Dialector初始化DB
func InitDB(dsn string, healthCheckInterval time.Duration, opts ...gorm.Option) (*gorm.DB, error) {
	dialector := AutoReconnectDialector{
		Dialector: mysql.Dialector{
			Config: &mysql.Config{
				DSN: dsn,
			},
		},
		healthCheckInterval: healthCheckInterval,
	}

	db, err := gorm.Open(dialector, opts...)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 关键连接池参数设置
	sqlDB.SetMaxIdleConns(10)                  // 空闲连接池中的最大连接数
	sqlDB.SetMaxOpenConns(100)                 // 数据库打开的最大连接数
	sqlDB.SetConnMaxLifetime(time.Hour)        // 连接可复用的最大时间
	sqlDB.SetConnMaxIdleTime(30 * time.Minute) // 连接最大空闲时间

	return db, nil
}

func InitReplicasDB(master string, replicas []string, healthCheckInterval time.Duration, opts ...gorm.Option) (*gorm.DB, error) {
	dialector := AutoReconnectDialector{
		Dialector: mysql.Dialector{
			Config: &mysql.Config{
				DSN: master,
			},
		},
		healthCheckInterval: healthCheckInterval,
	}
	// 初始化主库连接
	db, err := gorm.Open(dialector, opts...)
	if err != nil {
		return nil, err
	}
	// 配置读写分离插件
	rep := []gorm.Dialector{}
	for _, replica := range replicas {
		rep = append(rep, AutoReconnectDialector{
			Dialector: mysql.Dialector{
				Config: &mysql.Config{
					DSN: replica,
				},
			},
			healthCheckInterval: healthCheckInterval,
		})
	}
	err = db.Use(dbresolver.Register(dbresolver.Config{
		Sources:           []gorm.Dialector{dialector},   // 写操作源
		Replicas:          rep,                           // 读操作源
		Policy:            dbresolver.RoundRobinPolicy(), // 读操作负载均衡策略/ sources/replicas load balancing policy
		TraceResolverMode: true,                          // print sources/replicas mode in logger
	}))
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 关键连接池参数设置
	sqlDB.SetMaxIdleConns(10)                  // 空闲连接池中的最大连接数
	sqlDB.SetMaxOpenConns(100)                 // 数据库打开的最大连接数
	sqlDB.SetConnMaxLifetime(time.Hour)        // 连接可复用的最大时间
	sqlDB.SetConnMaxIdleTime(30 * time.Minute) // 连接最大空闲时间

	return db, nil
}
