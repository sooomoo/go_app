package services

import (
	"goapp/pkg/core"
	"goapp/pkg/distribute"
	"strings"

	"github.com/google/uuid"
)

// 此处的 id 长度不超过 53bit: 为了兼容js的 number
type IDService interface {
	NewUserID() int64
	NewOrderID() int64
	NewUUIDv7() string
	NewUUIDv8() core.UUIDv8
	NewSeqID() core.SeqID
}

type defaultIdService struct {
	userIdGenerator  *distribute.SnowflakeSecond
	orderIdGenerator *distribute.SnowflakeMillis
}

func NewDefaultIDService(workerId int64) IDService {
	// workerId 为 4bit 最多支持 16 台机器
	return &defaultIdService{
		// 每台机器每秒最多生成 32*4096 = 13,1072 个 id
		// 2^36 秒可以表示 2000 多年
		userIdGenerator: distribute.NewSnowflakeSecond(workerId, 5, 12),
		// 每台机器每毫秒最多生成 4096 个 id, 每秒可生成 32 * 1000 * 4096 = 1,3107,2000 个 id
		orderIdGenerator: distribute.NewSnowflakeMillis(workerId, 5, 12),
	}
}

func (i *defaultIdService) NewUserID() int64 {
	return i.userIdGenerator.Next()
}

func (i *defaultIdService) NewOrderID() int64 {
	return i.orderIdGenerator.Next()
}

func (i *defaultIdService) NewUUIDv7() string {
	val, err := uuid.NewV7()
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(val.String(), "-", "")
}

func (i *defaultIdService) NewUUIDv8() core.UUIDv8 {
	return core.NewUUIDv8()
}

func (i *defaultIdService) NewSeqID() core.SeqID {
	return core.NewSeqID()
}
