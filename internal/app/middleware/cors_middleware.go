package middleware

import (
	"goapp/internal/app/global"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CorsMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     global.GetAppConfig().Cors.AllowOrigins,
		AllowMethods:     global.GetAppConfig().Cors.AllowMethods,
		AllowHeaders:     global.GetAppConfig().Cors.AllowHeaders,
		ExposeHeaders:    global.GetAppConfig().Cors.ExposeHeaders,
		AllowCredentials: global.GetAppConfig().Cors.AllowCredentials,
		MaxAge:           time.Duration(global.GetAppConfig().Cors.MaxAge) * time.Minute,
		AllowWebSockets:  global.GetAppConfig().Cors.AllowWebSockets,
	})
}
