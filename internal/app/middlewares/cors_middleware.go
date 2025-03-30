package middlewares

import (
	"goapp/internal/app/global"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CorsMiddleware() gin.HandlerFunc {
	return cors.New(global.AppConfig.Cors)
}
