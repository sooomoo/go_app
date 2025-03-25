package app

import (
	"goapp/internal/app/handlers"
	"goapp/internal/app/hubs"
	"goapp/internal/app/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

func Run() {
	niu.InitSignHeaders("niu")
	err := hubs.Start()
	if err != nil {
		panic("hub start fail")
	}

	r := gin.Default()
	r.Use(middlewares.ReplayMiddleware)
	r.Use(middlewares.AuthorizeMiddleware)
	r.GET("/chat", hubs.UpgradeWebSocket)
	r.Use(niu.SignatureMiddleware(nil, niu.DefaultSignRule))

	handlers.RegisterRouters(r)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
