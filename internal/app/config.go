package app

type AppConfig struct {
	// 数据库配置
	Db    DBConfig
	Cache CacheConfig
}

type DBConfig struct {
	ConnectString string
}

type CacheConfig struct {
	Addr              string
	LockerAddr        string
	QueueAddr         string
	AuthenticatorAddr string
}
