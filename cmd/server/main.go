package main

import (
	"context"
	"goapp/internal/app"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/pprof"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.TODO()
	err := app.InitGlobalInstances(ctx, "", "", "")
	if err != nil {
		panic(err)
	}

	//logger, _ := zap.NewDevelopment(zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))

	store := cookie.NewStore([]byte("secret"))

	r := gin.New()
	r.Use(gin.LoggerWithWriter(os.Stdout), gin.RecoveryWithWriter(os.Stdout))

	pprof.RouteRegister(r, "debug/pprof")

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://foo.com"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return origin == "https://github.com"
		},
		MaxAge: 12 * time.Hour,
	}))

	r.Use(gzip.Gzip(gzip.DefaultCompression))

	r.Use(sessions.Sessions("mysession", store))
	// r.Use(func(ctx *gin.Context) {
	// 	ctx.SetSameSite(http.SameSiteLaxMode)
	// 	ctx.SetCookie("x-csrf-token", "123", 0, "/", "", false, false)
	// })
	r.GET("/forms2", func(c *gin.Context) {
		session := sessions.Default(c)
		v := session.Get("x-csrf-token")

		c.JSON(200, gin.H{
			"message": v,
		})
	})
	r.GET("/forms", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("x-csrf-token", "11111")
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("x-csrf-token", "22222", 0, "/", "", false, false)
		session.Save()
		c.JSON(200, gin.H{
			"message": "hello",
		})
	})

	// Example when panic happen.
	r.GET("/panic", func(c *gin.Context) {
		panic("An unexpected error happen!")
	})
	// r.Use(middlewares.ReplayMiddleware)
	// r.Use(middlewares.AuthorizeMiddleware)
	// r.GET("/chat", hubs.UpgradeChatWebSocket)
	// r.Use(niu.SignatureMiddleware(nil, niu.DefaultSignRule))

	// handlers.RegisterRouters(r)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
