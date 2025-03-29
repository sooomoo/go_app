package global

// 全局实例，用于全局使用，如缓存，消息队列，分布式锁，分布式ID生成器，分布式认证器，分布式消息总线等
import (
	"context"
	"fmt"
	"goapp/internal/app/hubs"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/panjf2000/ants/v2"
	"github.com/sooomo/niu"
)

var pool niu.RoutinePool

func GetPool() niu.RoutinePool { return pool }

var cache *niu.Cache

func GetCache() *niu.Cache { return cache }

var idGen *niu.DistributeId

func GetIdGenerator() *niu.DistributeId { return idGen }

var locker *niu.DistributeLocker

func GetLocker() *niu.DistributeLocker { return locker }

var queue niu.MessageQueue

func GetQueue() niu.MessageQueue { return queue }

var snowflake *niu.Snowflake

func GetSnowflake() *niu.Snowflake { return snowflake }

func GetChatHub() *niu.Hub { return hubs.GetChatHub() }

var authenticator *niu.Authenticator

func GetAuthenticator() *niu.Authenticator { return authenticator }

var corsConfig cors.Config

func Init(ctx context.Context) error {
	var err error = nil
	pool, err = ants.NewPool(500000, ants.WithExpiryDuration(5*time.Minute))
	if err != nil {
		return err
	}

	cache, err = niu.NewCacheWithAddr(ctx, "", "")
	if err != nil {
		return err
	}
	idGen, err = niu.NewDistributeId(ctx, cache.Master(), "idgen:user", 100000)
	if err != nil {
		return err
	}
	locker, err = niu.NewDistributeLockerWithAddr(ctx, "", 15*time.Second, niu.LinearRetryStrategy(2*time.Second))
	if err != nil {
		return err
	}

	queue, err = niu.NewRedisMessageQueueWithAddr(ctx, "", pool, 1024, 100)
	if err != nil {
		return err
	}
	snowflake = niu.NewSnowflake(1)

	authenticator, err = niu.NewAuthenticator(ctx, "",
		niu.NewHmacSignerResolver([]byte("")),
		niu.WithAllowMethods([]string{"GET", "POST", "OPTIONS"}),
		niu.WithAuthPaths([]string{}, []string{"/login", "/register"}),
		niu.WithJwt("niu.com", 2*time.Hour, []byte("")))

	mustAllowHeaders := authenticator.GetMustAllowHeaders()
	mustAllowHeaders = append(mustAllowHeaders, "Origin", "Content-Type", "X-CSRF-Token")
	corsConfig = cors.Config{
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     mustAllowHeaders,
		ExposeHeaders:    authenticator.GetMustExposeHeaders(),
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return true // origin == "https://github.com"
		},
		MaxAge: 2 * time.Hour,
	}

	err = hubs.StartChatHub(pool, []string{"niu-v1"})
	if err != nil {
		panic("hub start fail")
	}
	return err
}

func CorsMiddleware() gin.HandlerFunc {
	return cors.New(corsConfig)
}

func UpgradeChatWebSocket(c *gin.Context) {
	claims := GetAuthenticator().GetClaims(c)
	if claims == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	userId := fmt.Sprintf("%d", claims.UserId)
	err := GetChatHub().UpgradeWebSocket(userId, claims.Platform, c.Writer, c.Request)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}
