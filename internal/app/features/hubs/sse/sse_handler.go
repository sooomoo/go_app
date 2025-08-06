package sse

import (
	"github.com/gin-gonic/gin"
)

type SSEHandler struct{}

var (
	sseHandler *SSEHandler = &SSEHandler{}
)

func (s *SSEHandler) RegisterRoutes(router *gin.RouterGroup) {
	g := router.Group("/ai")
	g.GET("/ask", s.ask)
}

func (s *SSEHandler) ask(c *gin.Context) {
	GetAIHub().Serve(c)
}
