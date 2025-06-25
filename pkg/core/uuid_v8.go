package core

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type UUIDv8 [11]byte

var NilUUIDv8 = UUIDv8{}

// uuidV8Generator 用于生成适用于MySQL的UUID v8
type uuidV8Generator struct {
	mu            sync.Mutex
	lastTimestamp uint64
	counter       uint64
}

const uuidv8StartEpochMs = 1735660800000

// Generate 生成一个优化的UUID v8，格式为字节切片
func (g *uuidV8Generator) Generate() (UUIDv8, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 获取当前时间戳（毫秒）
	now := uint64(time.Now().UnixMilli() - uuidv8StartEpochMs)

	if now == g.lastTimestamp {
		// 同一毫秒内，增加计数器
		g.counter++
		// 检查计数器溢出
		if g.counter == 0 { // 当计数器达到65536时溢出回0
			// 等待下一毫秒
			for now <= g.lastTimestamp {
				time.Sleep(time.Microsecond * 10)
				now = uint64(time.Now().UnixMilli() - uuidv8StartEpochMs)
			}
		}
	} else if now < g.lastTimestamp {
		for now < g.lastTimestamp {
			time.Sleep(time.Microsecond * 10)
			now = uint64(time.Now().UnixMilli() - uuidv8StartEpochMs)
		}
		g.counter = 0
	} else {
		// 时间戳更新，重置计数器
		g.counter = 0
	}
	g.lastTimestamp = now

	// 构建UUID各部分
	uuid := UUIDv8{}

	// 7字节：56位：42 位时间戳+14 位计数
	pre := now<<14 + g.counter

	// 时间戳及计数部分（56位）
	binary.BigEndian.PutUint64(uuid[0:8], pre<<8) // 高48位为时间戳
	// 随机数部分（32位）
	_, err := rand.Read(uuid[7:11])
	if err != nil {
		return NilUUIDv8, errors.New("failed to generate random bytes")
	}

	return uuid, nil
}

var uuidv8Gen = &uuidV8Generator{
	lastTimestamp: 0, // 毫秒级时间戳
}

func NewUUIDv8() UUIDv8 {
	id, err := uuidv8Gen.Generate()
	if err != nil {
		return NilUUIDv8
	}
	return id
}

func (u UUIDv8) IsEmpty() bool {
	return len(u) == 0
}

// ToString 将UUID字节切片转换为标准字符串格式
func (u UUIDv8) String() string {
	return hex.EncodeToString(u[:])
}
