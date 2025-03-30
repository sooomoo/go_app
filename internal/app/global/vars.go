package global

import (
	"goapp/internal/app/config"

	"github.com/sooomo/niu"
	"gorm.io/gorm"
)

var Pool niu.CoroutinePool

var Cache *niu.Cache

var DistributeId *niu.DistributeId

var Locker *niu.DistributeLocker

var Queue niu.MessageQueue

var Snowflake *niu.Snowflake

var ChatHub *niu.Hub

var AppConfig *config.AppConfig

var Db *gorm.DB
