package main

import (
	"context"
	"goapp/internal/app"
	"goapp/internal/app/handlers"
	"goapp/internal/app/middlewares"
	"os"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

func main() {
	err := app.Init(context.TODO())
	if err != nil {
		panic(err)
	}

	r := gin.New()
	v1 := r.Group("/v1")
	v1.Use(gin.LoggerWithWriter(os.Stdout), gin.RecoveryWithWriter(os.Stdout))

	pprof.RouteRegister(r, "debug/pprof")

	v1.Use(middlewares.CorsMiddleware())
	v1.Use(middlewares.GzipMiddleware())
	v1.Use(middlewares.AuthenticateMiddleware())
	handlers.RegisterHandlers(v1)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
