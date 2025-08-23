package db

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/extra/bunexp"
)

func OpenDB(dsn string, replicas []string, healthCheckInterval time.Duration, opts ...pgdriver.Option) (*bun.DB, error) {
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

	// 读写分离
	repdbOpts := []bun.DBOption{bun.WithDiscardUnknownColumns()}
	for _, v := range replicas {
		optArr := []pgdriver.Option{
			pgdriver.WithDSN(v),
			pgdriver.WithTimeout(30 * time.Second),
		}

		optArr = append(optArr, opts...)
		repSqlDB := sql.OpenDB(pgdriver.NewConnector(optArr...))
		repSqlDB.SetMaxIdleConns(10)                  // 空闲连接池中的最大连接数
		repSqlDB.SetMaxOpenConns(50)                  // 数据库打开的最大连接数
		repSqlDB.SetConnMaxLifetime(10 * time.Minute) // 连接可复用的最大时间
		repSqlDB.SetConnMaxIdleTime(10 * time.Minute) // 连接最大空闲时间
		if err := repSqlDB.Ping(); err != nil {
			log.Fatal("Database connection failed:", err)
			return nil, err
		}
		repDB := bunexp.NewReadWriteConnResolver(bunexp.WithDBReplica(repSqlDB, bunexp.DBReplicaReadOnly))
		bun.WithConnResolver(repDB)
		repdbOpts = append(repdbOpts, bun.WithConnResolver(repDB))
	}

	db := bun.NewDB(sqlDB, pgdialect.New(), repdbOpts...)
	db.SetConnMaxIdleTime(10 * time.Minute)
	db.SetMaxIdleConns(10)                  // 空闲连接池中的最大连接数
	db.SetMaxOpenConns(50)                  // 数据库打开的最大连接数
	db.SetConnMaxLifetime(10 * time.Minute) // 连接可复用的最大时间
	db.SetConnMaxIdleTime(10 * time.Minute) // 连接最大空闲时间
	db.AddQueryHook(bundebug.NewQueryHook(
		// // disable the hook
		// bundebug.WithEnabled(false),

		// // BUNDEBUG=1 logs failed queries
		// // BUNDEBUG=2 logs all queries
		// bundebug.FromEnv("BUNDEBUG"),
		bundebug.WithVerbose(os.Getenv("env") == "dev"),
		bundebug.WithWriter(log.New(os.Stdout, "\r\n", log.LstdFlags).Writer()),
	))
	// db.AddQueryHook()
	return db, nil
}
