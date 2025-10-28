package main

import (
	"context"
	"goapp/internal/app/features"
	"goapp/internal/app/features/hubs/chat"
	"goapp/internal/app/features/hubs/sse"
	"goapp/internal/app/features/third"
	"goapp/internal/app/global"
	"goapp/internal/app/middleware"
	"goapp/pkg/ids"
	"net/http"
	"os"

	"github.com/gin-gonic/autotls"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

func main() {
	env := os.Getenv("env")
	log.Info().Msgf("server starting... runnint in [ %s ] mode", env)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	// 设置Gin模式
	switch env {
	case "release":
		gin.SetMode(gin.ReleaseMode)
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "test":
		gin.SetMode(gin.TestMode)
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// 如果使用雪花算法生成 ID 的话
	// 需要初始化雪花算法的节点 ID
	err := ids.IDSetNodeIDFromEnv("node_id")
	if err != nil {
		log.Error().Stack().Err(err).Msg("无法设置节点 ID")
		panic("无法设置节点 ID")
	}

	ctx := context.Background()
	global.Init(ctx) // 初始化全局变量, 失败时会 panic
	defer global.Release()

	r := gin.New()
	r.Use(gin.RecoveryWithWriter(os.Stdout))
	r.Use(middleware.LogMiddleware())
	r.Use(middleware.CorsMiddleware())

	if env == "dev" {
		pprof.RouteRegister(r, "debug/pprof")
	}

	// 第三方注册
	thirdGroup := r.Group("/third")
	{
		third.RegisterRoutes(thirdGroup)
	}

	// Hub 相关配置：hub 不需要与 api 一样的版本管理（v1）
	// 它可以通过 subprotocols 来管理版本
	hubGroup := r.Group("/hub")
	{
		hubGroup.Use(middleware.AuthMiddleware())
		chat.RegisterHubs(hubGroup)
	}

	// SSE 相关配置
	s := r.Group("/sse")
	{
		s.Use(middleware.GzipMiddleware())
		// s.Use(middleware.ReplayMiddleware())
		s.Use(middleware.AuthMiddleware())
		sse.RegisterHubs(s)
	}

	// 普通 API 请求相关配置
	v1 := r.Group("/v1")
	{
		v1.Use(middleware.GzipMiddleware())
		v1.Use(middleware.ReplayMiddleware())
		v1.Use(middleware.AuthMiddleware())
		v1.Use(middleware.SignMiddleware())
		v1.Use(middleware.CryptoMiddleware())
		features.RegisterRoutes(v1)
	}

	if env == "dev" {
		err = r.RunTLS(global.GetAppConfig().Addr, "E:\\experiment\\certs\\localhost+2.pem", "E:\\experiment\\certs\\localhost+2-key.pem")
	} else {
		// Start HTTPS server with automatic Let's Encrypt certificate management and HTTP-to-HTTPS redirection.
		// The server runs until interrupted and shuts down gracefully.
		err = autotls.Run(r, global.GetAppConfig().Domains...)
	}
	if err != nil && err != http.ErrServerClosed {
		log.Fatal().Stack().Err(err).Msg("启动服务器失败")
	}
}
