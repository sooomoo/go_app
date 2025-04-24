package hub

import (
	"goapp/pkg/core"
	"io"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// 客户端连接的消息
type LineMessage struct {
	UserId   string
	Platform core.Platform
	LineId   string
	Data     []byte
}

// 客户端连接的错误
type LineError struct {
	UserId   string
	Platform core.Platform
	LineId   string
	Error    error
}

// 客户端连接
type Line struct {
	hub        *Hub
	conn       *websocket.Conn
	id         string
	userId     string
	platform   core.Platform
	lastActive int64
	closeChan  chan core.Empty
	writeChan  chan []byte
}

func (ln *Line) Id() string { return ln.id }

func (ln *Line) UserId() string { return ln.userId }

func (ln *Line) Platform() core.Platform { return ln.platform }

func (ln *Line) LastActive() int64 { return atomic.LoadInt64(&ln.lastActive) }

func (ln *Line) Hub() *Hub { return ln.hub }

func (ln *Line) start() error {
	err := ln.hub.pool.Submit(func() {
		ln.conn.SetPingHandler(func(appData string) error {
			atomic.StoreInt64(&ln.lastActive, time.Now().Unix())
			return ln.conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(ln.hub.writeTimeout))
		})
		for {
			err := ln.conn.SetReadDeadline(time.Now().Add(ln.hub.readTimeout))
			if err != nil {
				ln.close(false, err)
				break
			}

			msgType, r, err := ln.conn.NextReader()
			if err != nil {
				ln.close(false, err)
				break
			}
			switch msgType {
			case websocket.CloseMessage:
				ln.close(false, nil)
				return
			case websocket.TextMessage:
				ln.close(true, nil) // 不允许文本消息
				return
			}

			// 池化读缓冲，提高性能
			buf := ln.hub.readBufferPool.Get()
			defer ln.hub.readBufferPool.Put(buf)
			_, err = io.Copy(buf, r)
			if err != nil {
				ln.close(false, err)
				return
			}

			atomic.StoreInt64(&ln.lastActive, time.Now().Unix())
			ln.hub.messageChan <- &LineMessage{ln.userId, ln.platform, ln.id, buf.Bytes()}
		}
	})
	if err != nil {
		ln.conn.Close()
		return err
	}

	err = ln.hub.pool.Submit(func() {
		for {
			select {
			case msg, ok := <-ln.writeChan:
				if ok {
					err = ln.conn.SetWriteDeadline(time.Now().Add(ln.hub.writeTimeout))
					if err != nil {
						ln.close(false, err)
						return
					}
					err = ln.conn.WriteMessage(websocket.BinaryMessage, msg)
					if err != nil {
						ln.close(false, err)
						return
					}
				}
			case _, ok := <-ln.closeChan:
				if ok {
					ln.close(true, nil)
					return
				}
			default:
				continue
			}
		}
	})
	if err != nil {
		ln.conn.Close()
		return err
	}

	ln.hub.registeredChanInternal <- ln
	return nil
}

func (ln *Line) close(sendCloseCtrl bool, err error) {
	if err != nil {
		ln.hub.errorChan <- &LineError{ln.userId, ln.platform, ln.id, err}
	}

	if sendCloseCtrl {
		// 需要调用以下消息发送关闭消息，这样客户端才能正确识别关闭代码
		// 否则会导致客户端一直重连
		// 不能调用 s.Conn.Close()
		message := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		ln.conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(ln.hub.writeTimeout))
	}
	ln.conn.Close()
	ln.hub.unregisteredChanInternal <- ln
}
