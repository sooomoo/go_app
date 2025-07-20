package sse

import (
	"errors"
	"fmt"
	"goapp/pkg/core"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type LineError struct {
	UserId   string
	Platform core.Platform
	LineId   string
	Error    error
}

// SSE 的单个连接
type Line struct {
	hub       *Hub
	writer    gin.ResponseWriter
	id        string
	userId    string
	platform  core.Platform
	extraData core.MapX

	writeChan chan string
	closeChan chan core.Empty

	isClosed atomic.Bool
}

func (ln *Line) push(msg string) error {
	_, err := fmt.Fprintf(ln.writer, "id: %d\ndata: %s\n\n", time.Now().UnixMicro(), msg)
	if err != nil {
		return err
	}
	ln.writer.Flush()
	return nil
}

func (ln *Line) start(c *gin.Context) {
	if ln.hub.isClosed.Load() {
		return
	}

	ln.hub.registeredChanInternal <- ln

	// 创建定时器用于心跳检测
	ticker := time.NewTicker(ln.hub.liveCheckDuration)
	defer ticker.Stop()

	// 发送初始连接确认
	ln.push("open")

	// 主循环处理消息
	for {
		if ln.isClosed.Load() || ln.hub.isClosed.Load() {
			return
		}

		select {
		case msg := <-ln.writeChan:
			// 发送消息到客户端
			ln.push(msg)
		case <-ticker.C:
			// 发送心跳保持连接
			ln.push("ping")
		case <-ln.closeChan:
			// 服务端主动关闭连接
			ln.isClosed.Store(true)
			ln.notifyClose()
			return
		case <-c.Request.Context().Done():
			// 客户端断开连接
			ln.close(errors.New("client disconnect"))
			return
		}
	}
}

func (ln *Line) close(err error) {
	if ln.isClosed.Load() {
		return
	}
	ln.isClosed.Store(true)

	if err != nil && !ln.hub.isClosed.Load() {
		ln.hub.errorChan <- &LineError{ln.userId, ln.platform, ln.id, err}
	}

	ln.notifyClose()
}

func (ln *Line) notifyClose() {
	if !ln.hub.isClosed.Load() {
		ln.hub.unregisteredChanInternal <- ln
	}
}
