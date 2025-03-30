package hubs

import "github.com/gin-gonic/gin"

func RegisterHubs(r *gin.RouterGroup) {
	hubGroup := r.Group("/hub")
	hubGroup.GET("/chat", upgradeChatWebSocket)
}
