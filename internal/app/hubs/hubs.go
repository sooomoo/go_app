package hubs

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

var chatHub *niu.Hub

func GetChatHub() *niu.Hub { return chatHub }

func StartChatHub(pool niu.RoutinePool, subprotocols []string) error {
	chatHub, err := niu.NewHub(
		subprotocols,
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
		return err
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
	return err
}

func UpgradeChatWebSocket(ctx *gin.Context) {
	userId := ctx.GetString("user_id")
	platform := ctx.GetInt("platform")
	err := chatHub.UpgradeWebSocket(
		userId,
		niu.Platform(platform),
		ctx.Writer,
		ctx.Request,
	)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
	}
}
