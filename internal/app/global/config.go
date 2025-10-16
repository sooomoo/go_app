package global

import (
	"goapp/pkg/db"

	"github.com/redis/go-redis/v9"
)

type AppConfig struct {
	Name          string              `mapstructure:"name"`
	Addr          string              `mapstructure:"addr"`
	Domains       []string            `mapstructure:"domains"`
	Id            string              `mapstructure:"id"`
	WorkerId      int64               `mapstructure:"worker_id"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Cache         CacheConfig         `mapstructure:"cache"`
	Locker        LockerConfig        `mapstructure:"locker"`
	Queue         QueueConfig         `mapstructure:"queue"`
	Hub           HubConfig           `mapstructure:"hub"`
	Authenticator AuthenticatorConfig `mapstructure:"authenticator"`
	Cors          CorsConfig          `mapstructure:"cors"`
}

type DatabaseConfig struct {
	Master   string    `mapstructure:"master"`
	Replicas []string  `mapstructure:"replicas"`
	Driver   db.Driver `mapstructure:"driver"`
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

type QueueConfig struct {
	Addr       string `mapstructure:"addr"`
	Db         int    `mapstructure:"db"`
	XAddMaxLen int    `mapstructure:"xadd_max_len"`
	BatchSize  int    `mapstructure:"batch_size"`
}

func (c *QueueConfig) GetRedisOption() *redis.Options {
	return &redis.Options{
		Addr: c.Addr,
		DB:   c.Db,
	}
}

type HubConfig struct {
	SubProtocols      []string `mapstructure:"sub_protocols"`
	LiveCheckDuration int64    `mapstructure:"live_check_duration"` // in second
	ConnMaxIdleTime   int64    `mapstructure:"conn_max_idle_time"`  // in second
	ReadTimeout       int64    `mapstructure:"read_timeout"`        // in second
	WriteTimeout      int64    `mapstructure:"write_timeout"`       // in second
	HandshakeTimeout  int64    `mapstructure:"handshake_timeout"`   // in second
	EnableCompression bool     `mapstructure:"enable_compression"`
}

type KeyPair struct {
	PublicKey  string `mapstructure:"pub"`
	PrivateKey string `mapstructure:"pri"`
}

type JwtConfig struct {
	Issuer             string `mapstructure:"issuer"`
	Secret             string `mapstructure:"secret"`
	AccessTtl          int64  `mapstructure:"access_ttl"`    // in minute
	RefreshTtl         int64  `mapstructure:"refresh_ttl"`   // in minute
	CookieDomain       string `mapstructure:"cookie_domain"` // 设置 cookie 时使用的域名，可以设置为 .niu.com， 这样 api.niu.com, img.niu.com 都能访问
	CookieSecure       bool   `mapstructure:"cookie_secure"`
	CookieHttpOnly     bool   `mapstructure:"cookie_httponly"`
	CookieSameSiteMode int    `mapstructure:"cookie_same_site_mode"`
}

type AuthenticatorConfig struct {
	BoxKeyPair        KeyPair   `mapstructure:"box_key_pair"`     // 用于加密和解密数据
	SignKeyPair       KeyPair   `mapstructure:"sign_key_pair"`    // 用于签名和验证数据
	EnableCrypto      bool      `mapstructure:"enable_crypto"`    // 是否启用加密
	PathsNeedCrypt    []string  `mapstructure:"paths_need_crypt"` // 如果包含*号，表示所有请求都是加密请求
	PathsNotCrypt     []string  `mapstructure:"paths_not_crypt"`  // 指定哪些请求不加密，优先级高于 PathsNeedCrypt
	PathsNeedAuth     []string  `mapstructure:"paths_need_auth"`  // 如果包含*号，表示所有请求都需要认证
	PathsNotAuth      []string  `mapstructure:"paths_not_auth"`   // 认证排除路径，优先级高于 PathsNeedAuth
	Jwt               JwtConfig `mapstructure:"jwt"`
	ReplayMaxInterval int64     `mapstructure:"replay_max_interval"` // in second，超过这个间隔时间的请求会被视为重放请求
}

type CorsConfig struct {
	AllowOrigins     []string `mapstructure:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers"`
	ExposeHeaders    []string `mapstructure:"expose_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int64    `mapstructure:"max_age"` // in minute
	AllowWebSockets  bool     `mapstructure:"allow_web_sockets"`
}
