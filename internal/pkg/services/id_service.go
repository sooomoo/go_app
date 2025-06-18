package services

import (
	"fmt"
	"sync"
	"time"
)

// 此处的 id 长度不超过 53bit: 为了兼容js的 number
type IDService interface {
	GenUserID() int64
	GenOrderID() int64
}

type defaultIdService struct {
	userIdGenerator  *secondIdGenerator
	orderIdGenerator *millisecondIdGenerator
}

func NewDefaultIDService(workerId int64) IDService {
	// workerId 为 4bit 最多支持 16 台机器
	return &defaultIdService{
		// 每台机器每秒最多生成 16*1024 = 16384 个 id
		// 2^38 秒可以表示 8000 多年
		userIdGenerator: newIdGenerator(workerId, 4, 10),
		// // 每台机器每毫秒最多生成 4096 个 id, 每秒可生成 32 * 1000 * 4096 = 1,3107,2000 个 id
		// orderIdGenerator: newMillisecondIdGenerator(workerId, 5, 12),
		// 每台机器每毫秒最多生成 4096 个 id, 每秒可生成 16 * 1000 * 1024 = 1638,4000 个 id
		// 2^38 毫秒可以表示 8.71 年，在这期间 js 的 number 是安全的
		orderIdGenerator: newMillisecondIdGenerator(workerId, 4, 10),
	}
}

func (i *defaultIdService) GenUserID() int64 {
	return i.userIdGenerator.Next()
}

func (i *defaultIdService) GenOrderID() int64 {
	return i.orderIdGenerator.Next()
}

type secondIdGenerator struct {
	epoch          int64 // id 起始时间
	workeridBits   int64 // 机器id所占位数
	sequenceBits   int64 // 序列所占的位数
	sequenceMask   int64 // 序列掩码
	timestampShift int64 // 时间戳的偏移

	sync.Mutex
	relativeTimestamp int64 // 上次生成id的时间
	workerid          int64 // 机器id
	sequence          int64 // 序列
}

const secondStartEpoch = 1747015868

func newIdGenerator(workerId, workerIdBits, sequenceBits int64) *secondIdGenerator {
	workeridMax := int64(-1 ^ (-1 << workerIdBits))
	if workerId < 0 || workerId > workeridMax {
		panic(fmt.Sprintf("workerid must be between 0 and %d", workeridMax))
	}
	sequenceMask := int64(-1 ^ (-1 << sequenceBits))
	timestampShift := workerIdBits + sequenceBits
	return &secondIdGenerator{
		epoch:          secondStartEpoch,
		workeridBits:   workerIdBits,
		sequenceBits:   sequenceBits,
		sequenceMask:   sequenceMask,
		timestampShift: timestampShift,
		workerid:       workerId,
		sequence:       0,
	}
}

func (s *secondIdGenerator) getRelativeTimestamp() int64 {
	return time.Now().Unix() - s.epoch
}

func (s *secondIdGenerator) Next() int64 {
	s.Lock()
	defer s.Unlock()

	now := s.getRelativeTimestamp()
	if now == s.relativeTimestamp {
		// 当同一时间戳（精度：秒）下多次生成id会增加序列号
		s.sequence = (s.sequence + 1) & s.sequenceMask
		if s.sequence == 0 {
			// 当前序列 Id 已经使用完，则需要等待下一秒
			for now <= s.relativeTimestamp {
				time.Sleep(time.Millisecond)
				now = s.getRelativeTimestamp()
			}
		}
	} else if now < s.relativeTimestamp {
		// 时钟回拨：等待到下一个时间戳
		for now < s.relativeTimestamp {
			time.Sleep(time.Millisecond)
			now = s.getRelativeTimestamp()
		}
	} else {
		// 不同时间戳（精度：秒）下直接使用序列号：0
		s.sequence = 0
	}

	s.relativeTimestamp = now

	return (now << s.timestampShift) | (s.workerid << s.sequenceBits) | s.sequence
}

type millisecondIdGenerator struct {
	epoch          int64 // id 起始时间
	workeridBits   int64 // 机器id所占位数
	sequenceBits   int64 // 序列所占的位数
	sequenceMask   int64 // 序列掩码
	timestampShift int64 // 时间戳的偏移

	sync.Mutex
	relativeTimestamp int64 // 上次生成id的时间
	workerid          int64 // 机器id
	sequence          int64 // 序列
}

const millisecondStartEpoch = 1747015868000

func newMillisecondIdGenerator(workerId, workerIdBits, sequenceBits int64) *millisecondIdGenerator {
	workeridMax := int64(-1 ^ (-1 << workerIdBits))
	if workerId < 0 || workerId > workeridMax {
		panic(fmt.Sprintf("workerid must be between 0 and %d", workeridMax))
	}
	sequenceMask := int64(-1 ^ (-1 << sequenceBits))
	timestampShift := workerIdBits + sequenceBits
	return &millisecondIdGenerator{
		epoch:          millisecondStartEpoch,
		workeridBits:   workerIdBits,
		sequenceBits:   sequenceBits,
		sequenceMask:   sequenceMask,
		timestampShift: timestampShift,
		workerid:       workerId,
		sequence:       0,
	}
}

func (s *millisecondIdGenerator) getRelativeTimestamp() int64 {
	return time.Now().UnixMilli() - s.epoch
}

func (s *millisecondIdGenerator) Next() int64 {
	s.Lock()
	defer s.Unlock()

	now := s.getRelativeTimestamp()
	if now == s.relativeTimestamp {
		// 当同一时间戳（精度：毫秒）下多次生成id会增加序列号
		s.sequence = (s.sequence + 1) & s.sequenceMask
		if s.sequence == 0 {
			// 当前序列 Id 已经使用完，则需要等待下一秒
			for now <= s.relativeTimestamp {
				time.Sleep(time.Microsecond * 10)
				now = s.getRelativeTimestamp()
			}
		}
	} else if now < s.relativeTimestamp {
		// 时钟回拨：等待到下一个时间戳
		for now < s.relativeTimestamp {
			time.Sleep(time.Microsecond * 10)
			now = s.getRelativeTimestamp()
		}
	} else {
		// 不同时间戳（精度：秒）下直接使用序列号：0
		s.sequence = 0
	}

	s.relativeTimestamp = now

	return (now << s.timestampShift) | (s.workerid << s.sequenceBits) | s.sequence
}
