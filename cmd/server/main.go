package main

import (
	"context"
	"goapp/internal/app"
	"goapp/internal/app/api/handlers"
	"goapp/internal/app/api/middleware"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

func main() {
	env := os.Getenv("env")
	ctx := context.Background()
	err := app.Init(ctx)
	if err != nil {
		panic(err)
	}

	// 设置Gin模式
	if env == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else if env == "test" {
		gin.SetMode(gin.TestMode)
	}

	r := gin.New()
	r.Use(middleware.CorsMiddleware())
	v1 := r.Group("/v1")
	v1.Use(gin.RecoveryWithWriter(os.Stdout), gin.LoggerWithWriter(os.Stdout))

	pprof.RouteRegister(r, "debug/pprof")

	v1.Use(middleware.LogMiddleware())
	v1.Use(middleware.GzipMiddleware())
	v1.Use(middleware.ReplayMiddleware())
	v1.Use(middleware.JwtMiddleware())
	v1.Use(middleware.SignMiddleware())
	v1.Use(middleware.CryptoMiddleware())
	handlers.RegisterRoutes(v1)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:    "127.0.0.1:8001",
		Handler: r,
	}
	// 优雅关闭
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	// 捕获终止信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务器...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("服务器强制关闭: %v", err)
	}

	log.Println("服务器已优雅地关闭")
}
