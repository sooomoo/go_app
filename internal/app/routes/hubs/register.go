package hubs

import "github.com/gin-gonic/gin"

var chatHub *ChatHub

func init() {
	chatHub = NewChatHub()
}

func GetChatHub() *ChatHub {
	return chatHub
}

func RegisterHubs(hubGroup *gin.RouterGroup) {
	chatHub.Start(hubGroup, "/chat")
}
