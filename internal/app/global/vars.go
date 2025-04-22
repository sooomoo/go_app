package global

import (
	"goapp/internal/app/config"
	"goapp/pkg/cache"
	"goapp/pkg/core"
	"goapp/pkg/distribute"
	"goapp/pkg/hub"

	"gorm.io/gorm"
)

var Pool core.CoroutinePool

var Cache *cache.Cache

var DistributeId *distribute.DistributeId

var UserIdGenerator *distribute.IdGenerator

var Locker *distribute.DistributeLocker

var Queue distribute.MessageQueue

var Snowflake *distribute.Snowflake

var ChatHub *hub.Hub

var AppConfig *config.AppConfig

var Db *gorm.DB
