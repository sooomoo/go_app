package app

import (
	"context"
	"goapp/internal/app/handlers"
	"goapp/internal/app/hubs"
	"goapp/internal/app/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

func Run() {
	ctx := context.TODO()
	err := InitGlobalInstances(ctx, "", "", "")
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.Use(middlewares.ReplayMiddleware)
	r.Use(middlewares.AuthorizeMiddleware)
	r.GET("/chat", hubs.UpgradeChatWebSocket)
	r.Use(niu.SignatureMiddleware(nil, niu.DefaultSignRule))

	handlers.RegisterRouters(r)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
