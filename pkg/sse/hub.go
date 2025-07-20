package sse

import (
	"errors"
	"fmt"
	"goapp/pkg/core"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type Hub struct {
	connections       sync.Map
	connCount         atomic.Int32
	pool              core.CoroutinePool
	liveCheckDuration time.Duration

	registeredChanInternal   chan *Line
	unregisteredChanInternal chan *Line
	registeredChan           chan *Line
	unregisteredChan         chan *Line
	errorChan                chan *LineError

	isClosed atomic.Bool
}

func NewHub(pool core.CoroutinePool, liveCheckDuration time.Duration) (*Hub, error) {
	if pool == nil {
		return nil, errors.New("pool must not nil")
	}
	if liveCheckDuration < time.Second {
		liveCheckDuration = time.Second
	}

	h := &Hub{
		connections:              sync.Map{},
		pool:                     pool,
		liveCheckDuration:        liveCheckDuration,
		registeredChanInternal:   make(chan *Line, 2048),
		unregisteredChanInternal: make(chan *Line, 2048),
		registeredChan:           make(chan *Line, 2048),
		unregisteredChan:         make(chan *Line, 2048),
		errorChan:                make(chan *LineError, 2048),
	}
	// 新的连接加入
	err := h.pool.Submit(func() {
		for ln := range h.registeredChanInternal {
			ln.closeChan = make(chan core.Empty)
			ln.writeChan = make(chan string, 2048)
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
			h.closeLineChans(ln)

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

func (h *Hub) Serve(c *gin.Context, userId string, platform core.Platform, lineId string, extraData core.MapX) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	// c.Header("Access-Control-Allow-Origin", "*")

	// 存下该平台新的连接
	ln := &Line{
		hub:       h,
		writer:    c.Writer,
		userId:    userId,
		platform:  core.Platform(platform),
		id:        lineId,
		extraData: extraData,
	}
	ln.start(c)
}

func (h *Hub) RegisteredChan() <-chan *Line { return h.registeredChan }

func (h *Hub) UnegisteredChan() <-chan *Line { return h.unregisteredChan }

func (h *Hub) ErrorChan() <-chan *LineError { return h.errorChan }

func (h *Hub) LiveCount() int { return int(h.connCount.Load()) }

func (h *Hub) closeLineChans(ln *Line) {
	if ln.closeChan != nil {
		close(ln.closeChan)
	}
	if ln.writeChan != nil {
		close(ln.writeChan)
	}

	ln.closeChan = nil
	ln.writeChan = nil
}

func (h *Hub) Close(wait time.Duration) {
	if h.isClosed.Load() {
		return
	}
	h.isClosed.Store(true)

	// 防止关闭时出现意外
	defer func() {
		r := recover()
		if r != nil {
			fmt.Println("close chan err")
		}
	}()

	uls := make([]*UserLines, 0)
	h.connections.Range(func(key, value any) bool {
		uls = append(uls, value.(*UserLines))
		return true
	})
	h.connections.Clear()
	// 关闭所有连接，并释放资源
	for _, v := range uls {
		v.Lock()
		defer v.Unlock()
		for _, ln := range v.lines {
			h.closeLineChans(ln)
		}
		v.lines = v.lines[:] // 清空
		v.lineCount.Store(0)
	}

	close(h.registeredChanInternal)
	close(h.unregisteredChanInternal)
	close(h.errorChan)
	close(h.registeredChan)
	close(h.unregisteredChan)
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

// 推送消息给指定用户的所有连接
func (h *Hub) PushMessage(userIds []string, data string) {
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

// 向用户指定的线路发送消息
func (h *Hub) PushToUserLines(userId string, data string, lineIds ...string) error {
	uls := h.GetUserLines(userId)
	if uls == nil {
		return errors.New("userlines empty")
	}
	uls.PushMessageToLines(data, lineIds...)
	return nil
}

// 广播消息
func (h *Hub) BroadcastMessage(data string) {
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
