package searcher

import "github.com/redis/go-redis/v9"

type WorkerConfig struct {
	Name     string         `mapstructure:"name"`
	Id       string         `mapstructure:"id"`
	Database DatabaseConfig `mapstructure:"database"`
	Cache    CacheConfig    `mapstructure:"cache"`
	Locker   LockerConfig   `mapstructure:"locker"`
	RMQ      RMQConfig      `mapstructure:"rmq"`
}

type DatabaseConfig struct {
	ConnectString string `mapstructure:"connect_string"`
}

type CacheConfig struct {
	Addr string `mapstructure:"addr"`
	Db   int    `mapstructure:"db"`
}

func (c *CacheConfig) GetRedisOption() *redis.Options {
	return &redis.Options{
		Addr: c.Addr,
		DB:   c.Db,
	}
}

type LockerConfig struct {
	Addr          string `mapstructure:"addr"`
	Db            int    `mapstructure:"db"`
	Ttl           int64  `mapstructure:"ttl"` // in second
	RetryStrategy string `mapstructure:"retry_strategy"`
	Backoff       int    `mapstructure:"backoff"`
	MaxRetry      int    `mapstructure:"max_retry"`
}

func (c *LockerConfig) GetRedisOption() *redis.Options {
	return &redis.Options{
		Addr: c.Addr,
		DB:   c.Db,
	}
}

type RMQConfig struct {
	Addr string `mapstructure:"addr"`
}
