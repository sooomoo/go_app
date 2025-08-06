package middleware

import (
	"goapp/internal/app"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CorsMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     app.GetGlobal().GetAppConfig().Cors.AllowOrigins,
		AllowMethods:     app.GetGlobal().GetAppConfig().Cors.AllowMethods,
		AllowHeaders:     app.GetGlobal().GetAppConfig().Cors.AllowHeaders,
		ExposeHeaders:    app.GetGlobal().GetAppConfig().Cors.ExposeHeaders,
		AllowCredentials: app.GetGlobal().GetAppConfig().Cors.AllowCredentials,
		MaxAge:           time.Duration(app.GetGlobal().GetAppConfig().Cors.MaxAge) * time.Minute,
		AllowWebSockets:  app.GetGlobal().GetAppConfig().Cors.AllowWebSockets,
	})
}
