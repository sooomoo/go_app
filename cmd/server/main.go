package main

import (
	"context"
	"goapp/internal/app/global"
	"goapp/internal/app/handlers"
	"os"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

func main() {
	err := global.Init(context.TODO())
	if err != nil {
		panic(err)
	}

	r := gin.New()
	r.Use(gin.LoggerWithWriter(os.Stdout), gin.RecoveryWithWriter(os.Stdout))

	pprof.RouteRegister(r, "debug/pprof")

	r.Use(global.CorsMiddleware())
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.POST("/login", handlers.HandleLogin)

	r.Use(global.GetAuthenticator().AuthenticateMiddleware)
	r.GET("/chat", global.UpgradeChatWebSocket)

	handlers.RegisterRouters(r)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
