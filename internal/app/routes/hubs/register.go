package hubs

import "github.com/gin-gonic/gin"

var chatHub *ChatHub

func GetChatHub() *ChatHub {
	return chatHub
}

func RegisterHubs(hubGroup *gin.RouterGroup) {
	chatHub = NewChatHub()
	chatHub.Start(hubGroup, "/chat")
}
