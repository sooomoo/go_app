package hubs

import (
	"errors"
	"fmt"
	"goapp/internal/app/config"
	"goapp/internal/app/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

var chatHub *niu.Hub

func StartChatHub(pool niu.CoroutinePool, config *config.HubConfig) (*niu.Hub, error) {
	var err error
	chatHub, err = niu.NewHub(
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
	svr := service.NewAuthService()
	claims := svr.GetClaims(c)
	if claims == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	userId := fmt.Sprintf("%d", claims.UserId)
	err := chatHub.UpgradeWebSocket(userId, claims.Platform, c.Writer, c.Request)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}
