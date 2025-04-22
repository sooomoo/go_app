package main

import (
	"context"
	"goapp/internal/app"
	"goapp/internal/app/global"
	"goapp/internal/app/routes/api"
	"goapp/internal/app/routes/hubs"
	"goapp/internal/app/routes/middleware"
	"goapp/pkg/core"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()
	err := app.Init(ctx)
	if err != nil {
		panic(err)
	}

	// 设置Gin模式
	env := os.Getenv("env")
	if env == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else if env == "test" {
		gin.SetMode(gin.TestMode)
	}

	r := gin.New()
	r.Use(func(ctx *gin.Context) {
		ctx.Next()
	})
	r.Use(gin.RecoveryWithWriter(os.Stdout))
	r.Use(middleware.LogMiddleware())
	r.Use(middleware.CorsMiddleware())

	if env == "dev" {
		pprof.RouteRegister(r, "debug/pprof")
	}

	// Hub 相关配置：hub 不需要与 api 一样的版本管理（v1）
	// 它可以通过 subprotocols 来管理版本
	hubGroup := r.Group("/hub")
	{
		hubGroup.Use(middleware.JwtMiddleware())
		hubs.RegisterHubs(hubGroup)
	}

	// 普通 API 请求相关配置
	v1 := r.Group("/v1")
	{
		v1.Use(middleware.GzipMiddleware())
		v1.Use(middleware.ReplayMiddleware())
		v1.Use(middleware.JwtMiddleware())
		v1.Use(middleware.SignMiddleware())
		v1.Use(middleware.CryptoMiddleware())
		api.RegisterRoutes(v1)
	}

	// 创建HTTP服务器
	svr := &http.Server{Addr: global.AppConfig.Addr, Handler: r}
	// 优雅关闭
	go func() {
		if err := svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	core.WaitSysSignal(func() {
		log.Println("正在关闭服务器...")
		c, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := svr.Shutdown(c); err != nil {
			log.Fatalf("服务器强制关闭: %v", err)
		}

		// 释放资源
		app.Release()
		log.Println("服务器已优雅地关闭")
	})
}
