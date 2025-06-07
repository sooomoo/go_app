package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
}

var (
	adminHandler *AdminHandler = &AdminHandler{}
)

func (h *AdminHandler) RegisterRoutes(router *gin.RouterGroup) {
	adminGroup := router.Group("/admin", func(c *gin.Context) {
		if c.Request.Header.Get("Authorization") != "foobar" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		// TODO: 此处还需要验证该用户的角色
		c.Next()
	})

	adminGroup.GET("/users", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "hello",
		})
	})
}
