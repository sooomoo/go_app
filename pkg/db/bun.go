package db

import (
	"context"
	"database/sql"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgerrcode"
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

var bunDB *bun.DB
var bunDBMutex sync.Mutex = sync.Mutex{}

func Get() *bun.DB {
	return bunDB
}

// MustOpenDB panic if OpenDB failed.
func MustOpenDB(dsn string, replicas []string, healthCheckInterval time.Duration, opts ...pgdriver.Option) *bun.DB {
	bunDBMutex.Lock()
	defer bunDBMutex.Unlock()

	if bunDB != nil {
		bunDB.Close()
		bunDB = nil
	}
	db, err := OpenDB(dsn, replicas, healthCheckInterval, opts...)
	if err != nil {
		panic(err)
	}
	bunDB = db

	return bunDB
}

// ToPGError tries to convert the given error to *pgdriver.Error.
//
// The second return value indicates whether the conversion was successful.
func ToPGError(err error) (*pgdriver.Error, bool) {
	if err == nil {
		return nil, false
	}

	pgErr, ok := err.(pgdriver.Error)
	return &pgErr, ok
}

// IsPGErrorCode reports whether the given error is a *pgdriver.Error
// with the given PostgreSQL error code.
func IsPGErrorCode(err error, code string) bool {
	er, ok := ToPGError(err)
	if !ok {
		return false
	}
	return er.Field('C') == code
}

func IsPGInvalidTransactionState(err error) bool {
	return IsPGErrorCode(err, pgerrcode.InvalidTransactionState)
}

// T 必须为结构体，不能是指针
func Update[T any](ctx context.Context, fn func(q *bun.UpdateQuery)) (sql.Result, error) {
	var model T
	q := Get().NewUpdate().Model(&model)
	fn(q)
	return q.Exec(ctx)
}

// T 必须为结构体，不能是指针
func Select[T any](ctx context.Context, desc T, fn func(q *bun.SelectQuery)) error {
	q := Get().NewSelect().Model(desc)
	fn(q)
	return q.Scan(ctx)
}

// T 必须为结构体，不能是指针
func Insert[T any](ctx context.Context, src T) error {
	_, err := Get().NewInsert().Model(src).Exec(ctx)
	return err
}

// T 必须为结构体，不能是指针
func Delete[T any](ctx context.Context, fn func(q *bun.DeleteQuery)) error {
	var model T
	d := Get().NewDelete().Model(&model)
	fn(d)
	_, err := d.Exec(ctx)
	return err
}

type BaseModelCreate struct {
	bun.BaseModel
	CreatedAt time.Time `bun:"created_at,notnull" json:"createdAt"`
}

type BaseModelCreateUpdate struct {
	bun.BaseModel
	CreatedAt time.Time `bun:"created_at,notnull" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull" json:"updatedAt"`
}

type BaseModelCreateUpdateDelete struct {
	bun.BaseModel
	CreatedAt time.Time `bun:"created_at,notnull" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull" json:"updatedAt"`
	DeletedAt time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deletedAt"`
}

type ListResult[T any] struct {
	Total int `json:"total"`
	Items []T `json:"items"`
}
