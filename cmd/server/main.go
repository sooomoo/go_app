package main

import (
	"context"
	"goapp/internal/app/global"
	"goapp/internal/app/handlers"
	"net/http"
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
	r.Use(global.GetAuthenticator().AuthenticateMiddleware)

	r.GET("/forms", func(c *gin.Context) {
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("x-csrf-token", "22222", 0, "/", "", false, false)
		c.JSON(200, gin.H{
			"message": "hello",
		})
	})

	r.GET("/chat", global.UpgradeChatWebSocket)

	handlers.RegisterRouters(r)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
