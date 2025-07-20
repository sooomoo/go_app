package hub

import (
	"goapp/pkg/core"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// 用户在各个平台的所有连接
type UserLines struct {
	sync.RWMutex
	lines     []*Line
	lineCount atomic.Int32
}

// 获取连接数量
func (u *UserLines) Len() int {
	return int(u.lineCount.Load())
}

// 添加连接
func (u *UserLines) add(line *Line) {
	u.Lock()
	defer u.Unlock()

	u.lines = append(u.lines, line)
	u.lineCount.Add(1)
}

// 删除连接
func (u *UserLines) remove(lineId string) {
	u.Lock()
	defer u.Unlock()
	lines := make([]*Line, 0)
	for _, v := range u.lines {
		if v.id != lineId {
			lines = append(lines, v)
		}
	}
	u.lines = lines
	u.lineCount.Add(-1)
}

// 获取指定连接
func (u *UserLines) Get(lineId string) *Line {
	u.RLock()
	defer u.RUnlock()

	for _, v := range u.lines {
		if v.id == lineId {
			return v
		}
	}
	return nil
}

// 获取指定平台的所有连接
func (u *UserLines) GetPlatformLines(platforms ...core.Platform) []*Line {
	if len(platforms) == 0 {
		return nil
	}

	u.RLock()
	defer u.RUnlock()

	lines := make([]*Line, 0)
	for _, v := range u.lines {
		if slices.Contains(platforms, v.platform) {
			lines = append(lines, v)
		}
	}
	return lines
}

// 关闭指定平台的所有连接
func (u *UserLines) ClosePlatforms(platforms ...core.Platform) {
	if len(platforms) == 0 {
		return
	}
	u.RLock()
	defer u.RUnlock()

	for _, line := range u.lines {
		if line.isClosed.Load() || line.closeChan == nil {
			continue
		}
		if slices.Contains(platforms, line.platform) {
			line.closeChan <- core.Empty{}
		}
	}
}

// 关闭除指定平台外的所有连接：切记，此处只发关闭命令，不处理其他逻辑
func (u *UserLines) ClosePlatformsExcept(exceptPlatforms ...core.Platform) {
	if len(exceptPlatforms) == 0 {
		return
	}

	u.RLock()
	defer u.RUnlock()

	for _, line := range u.lines {
		if line.isClosed.Load() || line.closeChan == nil {
			continue
		}
		if !slices.Contains(exceptPlatforms, line.platform) {
			line.closeChan <- core.Empty{}
		}
	}
}

// 关闭指定连接：切记，此处只发关闭命令，不处理其他逻辑
func (u *UserLines) CloseLines(lineIds ...string) {
	if len(lineIds) == 0 {
		return
	}

	u.RLock()
	defer u.RUnlock()

	for _, line := range u.lines {
		if line.isClosed.Load() || line.closeChan == nil {
			continue
		}
		if slices.Contains(lineIds, line.id) {
			line.closeChan <- core.Empty{}
		}
	}
}

// 关闭除指定连接外的所有连接
func (u *UserLines) CloseLinesExcept(exceptLineIds ...string) {
	if len(exceptLineIds) == 0 {
		return
	}

	u.RLock()
	defer u.RUnlock()

	for _, line := range u.lines {
		if !slices.Contains(exceptLineIds, line.id) {
			if line.isClosed.Load() || line.closeChan == nil {
				continue
			}
			line.closeChan <- core.Empty{}
		}
	}
}

// 关闭所有超过指定时间未活跃的连接
func (u *UserLines) closeInactiveLines(maxIdleSeconds int64) {
	if maxIdleSeconds <= 0 {
		return
	}

	u.RLock()
	defer u.RUnlock()

	for _, line := range u.lines {
		if time.Now().Unix()-line.lastActive > maxIdleSeconds {
			if line.isClosed.Load() || line.closeChan == nil {
				continue
			}
			line.closeChan <- core.Empty{}
		}
	}
}

// 关闭所有连接
func (u *UserLines) CloseAll() {
	u.RLock()
	defer u.RUnlock()

	for _, line := range u.lines {
		if line.isClosed.Load() || line.closeChan == nil {
			continue
		}
		line.closeChan <- core.Empty{}
	}
}

// 向该用户的所有连接发送消息
func (u *UserLines) PushMessage(data []byte) {
	if len(data) == 0 {
		return
	}

	u.RLock()
	defer u.RUnlock()

	for _, line := range u.lines {
		if line.isClosed.Load() || line.hub.isClosed.Load() || line.writeChan == nil {
			continue
		}
		line.writeChan <- data
	}
}

// 向该用户的所有连接发送消息，除了指定平台
func (u *UserLines) PushMessageExceptPlatforms(data []byte, exceptPlatforms ...core.Platform) {
	if len(exceptPlatforms) == 0 || len(data) == 0 {
		return
	}

	u.RLock()
	defer u.RUnlock()

	for _, line := range u.lines {
		if line.isClosed.Load() || line.hub.isClosed.Load() || line.writeChan == nil {
			continue
		}
		if slices.Contains(exceptPlatforms, line.platform) {
			continue
		}
		line.writeChan <- data
	}
}

// 向该用户的所有连接发送消息，除了指定连接
func (u *UserLines) PushMessageExceptLines(data []byte, exceptLineIds ...string) {
	if len(exceptLineIds) == 0 || len(data) == 0 {
		return
	}

	u.RLock()
	defer u.RUnlock()

	for _, line := range u.lines {
		if line.isClosed.Load() || line.hub.isClosed.Load() || line.writeChan == nil {
			continue
		}
		if slices.Contains(exceptLineIds, line.id) {
			continue
		}
		line.writeChan <- data
	}
}

// 向该用户的指定平台发送消息
func (u *UserLines) PushMessageToPlatforms(data []byte, platforms ...core.Platform) {
	if len(platforms) == 0 || len(data) == 0 {
		return
	}

	u.RLock()
	defer u.RUnlock()

	for _, line := range u.lines {
		if line.isClosed.Load() || line.hub.isClosed.Load() || line.writeChan == nil {
			continue
		}
		if slices.Contains(platforms, line.platform) {
			line.writeChan <- data
		}
	}
}

// 向该用户的指定连接发送消息
func (u *UserLines) PushMessageToLines(data []byte, lineIds ...string) {
	if len(lineIds) == 0 || len(data) == 0 {
		return
	}

	u.RLock()
	defer u.RUnlock()

	for _, line := range u.lines {
		if line.isClosed.Load() || line.hub.isClosed.Load() || line.writeChan == nil {
			continue
		}
		if slices.Contains(lineIds, line.id) {
			line.writeChan <- data
		}
	}
}
