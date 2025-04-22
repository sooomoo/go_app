package hubs

import "github.com/gin-gonic/gin"

func RegisterHubs(hubGroup *gin.RouterGroup) {
	hubGroup.GET("/chat", upgradeChatWebSocket)
}
