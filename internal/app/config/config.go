package config

import "github.com/gin-contrib/cors"

type AppConfig struct {
	Name     string
	Id       string
	WorkerId int64
	// 数据库配置
	Db            DBConfig
	Cache         CacheConfig
	Locker        LockerConfig
	Queue         QueueConfig
	Hub           HubConfig
	Authenticator AuthenticatorConfig
	Cors          cors.Config
}

type DBConfig struct {
	ConnectString string
}

type CacheConfig struct {
	Addr      string
	SlaveAddr string
}

type LockerConfig struct {
	Addr          string
	Ttl           int64 // in second
	RetryStrategy string
	Backoff       int
	MaxRetry      int
}

type QueueConfig struct {
	Addr       string
	XAddMaxLen int
	BatchSize  int
}

type HubConfig struct {
	SubProtocols []string
}

type AuthenticatorConfig struct {
	RedisAddr      string
	PathsNeedCrypt []string // 如果包含*号，表示所有请求都是加密请求
	PathsNotCrypt  []string // 指定哪些请求不加密，优先级高于 PathsNeedCrypt
	PathsNeedAuth  []string // 如果包含*号，表示所有请求都需要认证
	PathsNotAuth   []string // 认证排除路径，优先级高于 PathsNeedAuth
	Jwt            struct {
		Issuer     string
		Secret     string
		AccessTtl  int64 // in minute
		RefreshTtl int64 // in day
	}
}
