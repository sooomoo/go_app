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
	env := os.Getenv("env")
	configFile := ""
	if env == "prod" {
		configFile = ""
	}
	err := app.Init(context.TODO(), configFile)
	if err != nil {
		panic(err)
	}

	r := gin.New()
	v1 := r.Group("/v1")
	v1.Use(gin.LoggerWithWriter(os.Stdout), gin.RecoveryWithWriter(os.Stdout))

	pprof.RouteRegister(r, "debug/pprof")

	v1.Use(middlewares.CorsMiddleware())
	v1.Use(middlewares.GzipMiddleware())
	handlers.RegisterAuthHandlers(v1)

	r.Use(middlewares.AuthenticateMiddleware())
	handlers.RegisterHandlers(v1)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
