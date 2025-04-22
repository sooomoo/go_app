package middleware

import (
	"goapp/internal/app/global"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CorsMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     global.AppConfig.Cors.AllowOrigins,
		AllowMethods:     global.AppConfig.Cors.AllowMethods,
		AllowHeaders:     global.AppConfig.Cors.AllowHeaders,
		ExposeHeaders:    global.AppConfig.Cors.ExposeHeaders,
		AllowCredentials: global.AppConfig.Cors.AllowCredentials,
		MaxAge:           time.Duration(global.AppConfig.Cors.MaxAge) * time.Minute,
		AllowWebSockets:  global.AppConfig.Cors.AllowWebSockets,
	})
}
