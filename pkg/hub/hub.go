package hub

import (
	"errors"
	"fmt"
	"goapp/pkg/core"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Hub struct {
	connections sync.Map // key: userId , value: UserLines

	subprotocols []string
	connCount    atomic.Int32 // 所有仍在连接状态的数量
	pool         core.CoroutinePool

	liveCheckDuration  time.Duration
	liveTicker         *time.Ticker
	connMaxIdleSeconds int64
	upgrader           websocket.Upgrader

	writeTimeout   time.Duration
	readTimeout    time.Duration
	readBufferPool *core.ByteBufferPool

	messageChan              chan *LineMessage
	registeredChanInternal   chan *Line
	unregisteredChanInternal chan *Line
	registeredChan           chan *Line
	unregisteredChan         chan *Line
	errorChan                chan *LineError

	isClosed atomic.Bool
}

func NewHub(
	subprotocols []string,
	liveCheckDuration, connMaxIdleTime,
	readTimeout, writeTimeout time.Duration,
	pool core.CoroutinePool,
	handshakeTimeout time.Duration,
	enableCompression bool,
	checkOriginFn func(r *http.Request) bool,
) (*Hub, error) {
	if pool == nil {
		return nil, errors.New("pool must not nil")
	}
	if liveCheckDuration < time.Second {
		liveCheckDuration = time.Second
	}

	h := &Hub{
		connections:              sync.Map{},
		subprotocols:             subprotocols,
		pool:                     pool,
		liveCheckDuration:        liveCheckDuration,
		readTimeout:              readTimeout,
		writeTimeout:             writeTimeout,
		connMaxIdleSeconds:       int64(connMaxIdleTime),
		liveTicker:               time.NewTicker(liveCheckDuration),
		readBufferPool:           core.NewByteBufferPool(0, 2048),
		messageChan:              make(chan *LineMessage, 4096),
		registeredChanInternal:   make(chan *Line, 2048),
		unregisteredChanInternal: make(chan *Line, 2048),
		registeredChan:           make(chan *Line, 2048),
		unregisteredChan:         make(chan *Line, 2048),
		errorChan:                make(chan *LineError, 2048),
		upgrader: websocket.Upgrader{
			EnableCompression: enableCompression,
			HandshakeTimeout:  handshakeTimeout,
			ReadBufferSize:    4096,
			WriteBufferSize:   4096,
			Subprotocols:      subprotocols,
			WriteBufferPool:   &sync.Pool{},
			CheckOrigin:       checkOriginFn,
		},
	}

	// 检测连接可用性
	err := h.pool.Submit(func() {
		ticker := h.liveTicker
		for range ticker.C {
			delArr := make([]string, 0)
			h.connections.Range(func(key, value any) bool {
				conn := value.(*UserLines)
				conn.closeInactiveLines(h.connMaxIdleSeconds)
				if conn.Len() == 0 {
					delArr = append(delArr, key.(string))
				}
				return true
			})

			for _, v := range delArr {
				h.connections.Delete(v)
			}
		}
	})
	if err != nil {
		return nil, err
	}
	// 新的连接加入
	err = h.pool.Submit(func() {
		for ln := range h.registeredChanInternal {
			ln.closeChan = make(chan core.Empty)
			ln.writeChan = make(chan []byte, 2048)
			// 新的连接加入
			lines, _ := h.connections.LoadOrStore(ln.userId, &UserLines{lines: []*Line{}})
			lines.(*UserLines).add(ln)
			h.connCount.Add(1)

			h.registeredChan <- ln
		}
	})
	if err != nil {
		return nil, err
	}
	// 连接断开
	err = h.pool.Submit(func() {
		for ln := range h.unregisteredChanInternal {
			// 连接断开
			lines, ok := h.connections.Load(ln.userId)
			if !ok {
				continue // 没有找到相关信息
			}

			userLines := lines.(*UserLines)
			userLines.remove(ln.id)
			h.connCount.Add(-1)
			// 先删后关，防止在关闭之后，出现向通道意外发送的情况
			close(ln.closeChan)
			close(ln.writeChan)

			// 如果用户没有连接，则删除用户
			if ok && userLines.Len() == 0 {
				h.connections.Delete(ln.userId)
			}

			h.unregisteredChan <- ln
		}
	})
	if err != nil {
		return nil, err
	}

	return h, nil
}

// 返回只读通道
func (h *Hub) MessageChan() <-chan *LineMessage { return h.messageChan }

func (h *Hub) RegisteredChan() <-chan *Line { return h.registeredChan }

func (h *Hub) UnegisteredChan() <-chan *Line { return h.unregisteredChan }

func (h *Hub) ErrorChan() <-chan *LineError { return h.errorChan }

func (h *Hub) LiveCount() int { return int(h.connCount.Load()) }

func (h *Hub) Close(wait time.Duration) {
	if h.isClosed.Load() {
		return
	}
	h.isClosed.Store(true)

	if h.liveTicker != nil {
		h.liveTicker.Stop()
		h.liveTicker = nil
	}
	uls := make([]*UserLines, 0)
	h.connections.Range(func(key, value any) bool {
		uls = append(uls, value.(*UserLines))
		return true
	})
	h.connections.Clear()
	h.pool.Submit(func() {
		for _, v := range uls {
			v.CloseAll()
		}
	})

	time.Sleep(wait)
	defer func() {
		r := recover()
		if r != nil {
			fmt.Println("close chan err")
		}
	}()

	close(h.messageChan)
	close(h.registeredChanInternal)
	close(h.unregisteredChanInternal)
	close(h.errorChan)
	close(h.registeredChan)
	close(h.unregisteredChan)

	h.readBufferPool = nil
}

// 获取指定用户的所有连接
func (h *Hub) GetUserLines(userId string) *UserLines {
	lines, ok := h.connections.Load(userId)
	if !ok {
		return nil
	}
	return lines.(*UserLines)
}

// 获取指定用户的指定连接
func (h *Hub) GetUserLine(userId, lineId string) *Line {
	lines, ok := h.connections.Load(userId)
	if !ok {
		return nil
	}
	return lines.(*UserLines).Get(lineId)
}

// 关闭指定用户的所有连接
func (h *Hub) CloseUserLines(userIds ...string) {
	if len(userIds) == 0 {
		return
	}

	for _, userId := range userIds {
		if len(userId) == 0 {
			continue
		}
		lines, ok := h.connections.Load(userId)
		if !ok {
			continue
		}
		lines.(*UserLines).CloseAll()
	}
}

func (h *Hub) PushMessage(userIds []string, data []byte) {
	if len(userIds) == 0 || len(data) == 0 {
		return
	}
	h.pool.Submit(func() {
		for _, userId := range userIds {
			lines, ok := h.connections.Load(userId)
			if !ok {
				continue
			}
			lines.(*UserLines).PushMessage(data)
		}
	})
}

func (h *Hub) PushToUserLines(userId string, data []byte, lineIds ...string) error {
	uls := h.GetUserLines(userId)
	if uls == nil {
		return errors.New("userlines empty")
	}
	uls.PushMessageToLines(data, lineIds...)
	return nil
}

func (h *Hub) BroadcastMessage(data []byte) {
	if len(data) == 0 {
		return
	}
	h.pool.Submit(func() {
		h.connections.Range(func(key, lns any) bool {
			lns.(*UserLines).PushMessage(data)
			return true
		})
	})
}

func (h *Hub) UpgradeWebSocket(userId string, platform core.Platform, lineId string, extraData ExtraData, w http.ResponseWriter, r *http.Request) error {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	// 存下该平台新的连接
	ln := &Line{
		hub:        h,
		conn:       conn,
		userId:     userId,
		platform:   core.Platform(platform),
		id:         lineId,
		extraData:  extraData,
		lastActive: time.Now().Unix(),
	}

	// 开始监听该连接的消息
	return ln.start()
}
