package hubs

import (
	"errors"
	"fmt"
	"goapp/internal/app"
	"goapp/internal/app/services/headers"
	"goapp/pkg/bytes"
	"goapp/pkg/core"
	"goapp/pkg/hub"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
)

type ChatMsgType byte

const (
	ChatMsgTypeReady ChatMsgType = 1
	ChatMsgTypePing  ChatMsgType = 5
	ChatMsgTypePong  ChatMsgType = 6
)

type ChatRespCode byte

const (
	ChatRespCodeOk ChatRespCode = 1
)

type ChatHub struct {
	*hub.Hub

	pool   core.CoroutinePool
	config *app.HubConfig

	protooal *bytes.PacketProtocol
}

func NewChatHub() *ChatHub {
	return &ChatHub{
		pool:     app.GetGlobal().GetCoroutinePool(),
		config:   &app.GetGlobal().GetAppConfig().Hub,
		protooal: bytes.NewMsgPackProtocol(nil, nil),
	}
}

func (h *ChatHub) Start(router *gin.RouterGroup, path string) {
	hub, err := hub.NewHub(
		h.config.SubProtocols,
		time.Second*time.Duration(h.config.LiveCheckDuration),
		time.Second*time.Duration(h.config.ConnMaxIdleTime),
		time.Second*time.Duration(h.config.ReadTimeout),
		time.Second*time.Duration(h.config.WriteTimeout),
		h.pool,
		time.Second*time.Duration(h.config.HandshakeTimeout),
		h.config.EnableCompression,
		func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return slices.Contains(app.GetGlobal().GetAppConfig().Cors.AllowOrigins, origin)
		},
	)
	if err != nil {
		panic(err)
	}
	h.Hub = hub

	go h.listen()

	router.GET(path, h.upgrade)
}

func (h *ChatHub) listen() {
	for {
		select {
		case msg, ok := <-h.MessageChan():
			if ok {
				h.handleReceivedMsg(msg)
			}
		case r, ok := <-h.RegisteredChan():
			if ok {
				h.handleLineRegistered(r)
			}
		case u, ok := <-h.UnegisteredChan():
			if ok {
				h.handleLineUnegistered(u)
			}
		case e, ok := <-h.ErrorChan():
			if ok {
				h.handleLineError(e)
			}
		default:
			// 如果没有待处理的消息，休眠 10ms
			time.Sleep(time.Millisecond * 10)
		}
	}
}

func (h *ChatHub) upgrade(c *gin.Context) {
	if h.Hub == nil {
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
	userLines := h.GetUserLines(userId)
	if userLines != nil {
		userLines.CloseLines(clientId)
	}

	clientKeys := headers.GetClientKeys(c)
	extraData := hub.ExtraData{}
	extraData.Set(headers.KeyClientKeys, clientKeys)

	err := h.UpgradeWebSocket(userId, claims.Platform, clientId, extraData, c.Writer, c.Request)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func (h *ChatHub) handleReceivedMsg(msg *hub.LineMessage) {
	meta, err := h.protooal.GetMeta(msg.Data)
	if err != nil {
		// log err
		return
	}
	fmt.Printf("[HUB] receive msg: userid->%v, platform->%v, line->%v, msg type->%v, requestId->%v, timestamp(relative)->%v\n", msg.UserId, msg.Platform, msg.LineId, meta.MsgType, meta.RequestId, meta.GetConvertedTimestamp())
	if meta.MsgType == byte(ChatMsgTypePing) {
		resp, err := h.protooal.EncodeResp(int32(ChatMsgTypePong), meta.RequestId, byte(ChatRespCodeOk), nil)
		if err == nil {
			h.PushToUserLines(msg.UserId, resp, msg.LineId)
		}
	}
}

func (h *ChatHub) handleLineRegistered(r *hub.Line) {
	fmt.Printf("[HUB] line registered: userid->%v, platform->%v, line->%v\n", r.UserId(), r.Platform(), r.Id())
	resp, err := h.protooal.EncodeResp(int32(ChatMsgTypeReady), 0, byte(ChatRespCodeOk), nil)
	if err == nil {
		h.PushToUserLines(r.UserId(), resp, r.Id())
	}
}

func (h *ChatHub) handleLineUnegistered(u *hub.Line) {
	fmt.Printf("[HUB] line unregistered: userid->%v, platform->%v, line->%v\n", u.UserId(), u.Platform(), u.Id())
}

func (h *ChatHub) handleLineError(e *hub.LineError) {
	fmt.Printf("[HUB] line error: userid->%v, platform->%v, line->%v, err:%v\n", e.UserId, e.Platform, e.LineId, e.Error)
}
