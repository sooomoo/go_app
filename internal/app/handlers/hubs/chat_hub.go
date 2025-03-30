package hubs

import (
	"errors"
	"fmt"
	"goapp/internal/app/config"
	"goapp/internal/app/services"
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
		2*time.Minute,
		time.Minute,
		30*time.Second,
		30*time.Second,
		pool,
		10*time.Second,
		false,
		func(r *http.Request) bool { return true },
	)
	if err != nil {
		return nil, err
	}
	msgProto := niu.NewMsgPackProtocol(nil, nil)
	err = pool.Submit(func() {
		for {
			select {
			case msg, ok := <-chatHub.MessageChan():
				if ok {
					var mp map[string]any
					if msgType, err := msgProto.DecodeReq(msg.Data, &mp); err == nil {
						fmt.Printf("recv msg type:%v, val is:%v", msgType, mp)
					}
				}
			case r, ok := <-chatHub.RegisteredChan():
				if ok {
					fmt.Printf("line registered: userid->%v, platform->%v", r.UserId(), r.Platform())
				}
			case u, ok := <-chatHub.UnegisteredChan():
				if ok {
					fmt.Printf("line unregistered: userid->%v, platform->%v", u.UserId(), u.Platform())
				}
			case e, ok := <-chatHub.ErrorChan():
				if ok {
					fmt.Printf("line error: userid->%v, platform->%v, err:%v", e.UserId, e.Platform, e.Error)
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
	svr := services.NewAuthService()
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
