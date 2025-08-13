package sse

import (
	"errors"
	"fmt"
	"goapp/internal/app/global"
	"goapp/internal/app/shared/claims"
	"goapp/internal/app/shared/headers"
	"goapp/pkg/core"
	"goapp/pkg/sse"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AIHub struct {
	hub *sse.Hub

	pool core.CoroutinePool
}

func NewAIHub() (*AIHub, error) {
	pool := global.GetCoroutinePool()
	h, err := sse.NewHub(pool, 30*time.Second)
	if err != nil {
		panic(err)
	}
	return &AIHub{
		pool: pool,
		hub:  h,
	}, nil
}

func (h *AIHub) Serve(c *gin.Context) {
	if h.hub == nil {
		panic(errors.New("chat hub is nil"))
	}
	lineID := strings.TrimSpace(c.Query("id"))
	if len(lineID) == 0 {
		c.AbortWithError(401, errors.New("invalid id"))
		return
	}
	// 解析Token
	claims := claims.GetClaims(c)
	userId := fmt.Sprintf("%d", claims.UserId)

	// 关闭旧的连接，客户端自动重连时会携带之前用过的 id
	lines := aiHub.hub.GetUserLines(userId)
	if lines != nil {
		lines.CloseLines(lineID)
	}

	clientKeys := headers.GetClientKeys(c)
	extraData := core.MapX{}
	extraData.SetValue(headers.KeyClientKeys, clientKeys)
	h.hub.Serve(c, userId, claims.Platform, lineID, extraData)
}

func (h *AIHub) StartBroadcastTest() {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			h.hub.BroadcastMessage(fmt.Sprintf("hello world %d", time.Now().Unix()))
		}
	}()
}
