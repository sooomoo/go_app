package main

import (
	"context"
	"goapp/internal/workers/searcher"
	"goapp/pkg/core"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	env := os.Getenv("env")
	log.Info().Msgf("【worker】【searcher】 starting... runnint in [ %s ] mode", env)
	ctx := context.Background()
	searcher.GetGlobal().Init(ctx) // 初始化全局变量, 失败时会 panic

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

	log.Info().Msg("准备启动任务...")
	time.Sleep(time.Second * 5)

	log.Info().Msg("任务启动完毕！")

	// Wait system signal, and cleanup resources
	core.WaitSysSignal(func() {
		log.Info().Msg("正在关闭服务器...")
		c, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// 关闭所有任务

		// 释放资源
		searcher.GetGlobal().Release()

		<-c.Done()
		log.Info().Msg("服务器已优雅地关闭")
	})
}
