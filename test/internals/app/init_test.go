package features_test

import (
	"context"
	"goapp/internal/app/global"
	"goapp/pkg/ids"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	// 如果使用雪花算法生成 ID 的话
	// 需要初始化雪花算法的节点 ID
	err := ids.IDSetNodeIDFromEnv("node_id")
	if err != nil {
		panic("无法设置节点 ID")
	}

	env := os.Getenv("env")
	log.Info().Msgf("server starting... runnint in [ %s ] mode", env)
	ctx := context.Background()
	global.Init(ctx) // 初始化全局变量, 失败时会 panic
}
