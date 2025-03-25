package middlewares

import (
	"goapp/internal/app/services"

	"github.com/gin-gonic/gin"
)

func AuthorizeMiddleware(ctx *gin.Context) {
	authSvr := services.NewAuthService()
	code := authSvr.AuthorizeRequest(ctx)
	if code != 0 {
		ctx.AbortWithStatus(code)
		return
	}
	ctx.Next()
}
