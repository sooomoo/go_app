package hubs

import (
	"errors"
	"fmt"
	"goapp/internal/app/config"
	"goapp/internal/app/service/headers"
	"goapp/pkg/core"
	"goapp/pkg/hub"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var chatHub *hub.Hub

func StartChatHub(pool core.CoroutinePool, config *config.HubConfig) (*hub.Hub, error) {
	var err error
	chatHub, err = hub.NewHub(
		config.SubProtocols,
		time.Second*time.Duration(config.LiveCheckDuration),
		time.Second*time.Duration(config.ConnMaxIdleTime),
		time.Second*time.Duration(config.ReadTimeout),
		time.Second*time.Duration(config.WriteTimeout),
		pool,
		time.Second*time.Duration(config.HandshakeTimeout),
		config.EnableCompression,
		func(r *http.Request) bool { return true },
	)
	if err != nil {
		return nil, err
	}
	err = pool.Submit(func() {
		for {
			select {
			case msg, ok := <-chatHub.MessageChan():
				if ok {
					handleReceivedMsg(msg)
				}
			case r, ok := <-chatHub.RegisteredChan():
				if ok {
					handleLineRegistered(r)
				}
			case u, ok := <-chatHub.UnegisteredChan():
				if ok {
					handleLineUnegistered(u)
				}
			case e, ok := <-chatHub.ErrorChan():
				if ok {
					handleLineError(e)
				}
			default:
				continue
			}
		}
	})
	return chatHub, err
}

func upgradeChatWebSocket(c *gin.Context) {
	if chatHub == nil {
		panic(errors.New("chat hub is nil"))
	}
	// 解析Token
	claims := headers.GetClaims(c)
	userId := fmt.Sprintf("%d", claims.UserId)

	// clientId 全局唯一
	clientId := headers.GetClientId(c)
	if len(clientId) == 0 {
		c.AbortWithError(401, errors.New("invalid cid"))
		return
	}

	// 此处可以踢出其它不希望的连接：比如多个平台只允许一个连接
	// 此处指定为：同一个 client 仅允许一个连接
	userLines := chatHub.GetUserLines(userId)
	if userLines != nil {
		userLines.CloseLines(clientId)
	}

	clientKeys := headers.GetClientKeys(c)
	extraData := hub.ExtraData{}
	extraData.Set(headers.KeyClientKeys, clientKeys)

	err := chatHub.UpgradeWebSocket(userId, claims.Platform, clientId, extraData, c.Writer, c.Request)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}
