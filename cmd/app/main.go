package main

import (
	"context"
	"goapp/internal/app"
	"goapp/internal/app/routes/api"
	"goapp/internal/app/routes/hubs"
	"goapp/internal/app/routes/middleware"
	"goapp/internal/app/routes/third"
	"goapp/pkg/core"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	env := os.Getenv("env")
	log.Info().Msgf("server starting... runnint in [ %s ] mode", env)
	ctx := context.Background()
	app.GetGlobal().Init(ctx) // 初始化全局变量, 失败时会 panic

	// 设置Gin模式
	if env == "release" {
		gin.SetMode(gin.ReleaseMode)
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else if env == "test" {
		gin.SetMode(gin.TestMode)
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

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
		hubs.RegisterHubs(hubGroup)
	}

	// 普通 API 请求相关配置
	v1 := r.Group("/v1")
	{
		v1.Use(middleware.GzipMiddleware())
		v1.Use(middleware.ReplayMiddleware())
		v1.Use(middleware.AuthMiddleware())
		v1.Use(middleware.SignMiddleware())
		v1.Use(middleware.CryptoMiddleware())
		api.RegisterRoutes(v1)
	}

	// 创建HTTP服务器
	svr := &http.Server{Addr: app.GetGlobal().GetAppConfig().Addr, Handler: r}
	// 优雅关闭
	go func() {
		if env == "dev" {
			if err := svr.ListenAndServeTLS("/Users/muro/work/certs/cert.pem", "/Users/muro/work/certs/key.pem"); err != nil && err != http.ErrServerClosed {
				log.Fatal().Stack().Err(err).Msg("启动服务器失败")
			}
		} else {
			if err := svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal().Stack().Err(err).Msg("启动服务器失败")
			}
		}
	}()

	core.WaitSysSignal(func() {
		log.Info().Msg("正在关闭服务器...")
		c, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := svr.Shutdown(c); err != nil {
			log.Fatal().Stack().Err(err).Msg("服务器关闭失败")
		}

		// 释放资源
		app.GetGlobal().Release()

		<-c.Done()
		log.Info().Msg("服务器已优雅地关闭")
	})
}
