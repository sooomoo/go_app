package sse

import "github.com/gin-gonic/gin"

var aiHub *AIHub

func GetAIHub() *AIHub {
	return aiHub
}

func NewSSEHubs() {
	var err error
	aiHub, err = NewAIHub()
	if err != nil {
		panic(err)
	}
	aiHub.StartBroadcastTest()
}

func RegisterHubs(hubGroup *gin.RouterGroup) {
	NewSSEHubs()
	sseHandler.RegisterRoutes(hubGroup)
}
