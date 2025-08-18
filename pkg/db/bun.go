package db

import (
	"database/sql"
	"log"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func OpenDB(dsn string, healthCheckInterval time.Duration, opts ...pgdriver.Option) (*bun.DB, error) {
	optArr := []pgdriver.Option{
		pgdriver.WithDSN(dsn),
		pgdriver.WithTimeout(30 * time.Second),
	}

	optArr = append(optArr, opts...)
	sqlDB := sql.OpenDB(pgdriver.NewConnector(optArr...))
	sqlDB.SetMaxIdleConns(10)                  // 空闲连接池中的最大连接数
	sqlDB.SetMaxOpenConns(50)                  // 数据库打开的最大连接数
	sqlDB.SetConnMaxLifetime(10 * time.Minute) // 连接可复用的最大时间
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // 连接最大空闲时间
	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Database connection failed:", err)
		return nil, err
	}
	db := bun.NewDB(sqlDB, pgdialect.New(), bun.WithDiscardUnknownColumns())
	// db.AddQueryHook()
	return db, nil
}
