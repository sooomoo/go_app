package main

import (
	"context"
	"goapp/internal/app"
	"goapp/internal/app/api/handlers"
	"goapp/internal/app/api/middleware"
	"goapp/internal/app/global"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
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
	r.Use(middleware.CorsMiddleware())
	v1 := r.Group("/v1")
	v1.Use(gin.RecoveryWithWriter(os.Stdout), gin.LoggerWithWriter(os.Stdout))

	if env == "dev" {
		pprof.RouteRegister(r, "debug/pprof")
	}

	v1.Use(middleware.LogMiddleware())
	v1.Use(middleware.GzipMiddleware())
	v1.Use(middleware.ReplayMiddleware())
	v1.Use(middleware.JwtMiddleware())
	v1.Use(middleware.SignMiddleware())
	v1.Use(middleware.CryptoMiddleware())
	handlers.RegisterRoutes(v1)

	// 创建HTTP服务器
	svr := &http.Server{Addr: global.AppConfig.Addr, Handler: r}
	// 优雅关闭
	go func() {
		if err := svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	niu.WaitSysSignal(func() {
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
