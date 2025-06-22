package distribute

import (
	"fmt"
	"sync"
	"time"
)

type Snowflake interface {
	Next() int64
	GetWorkerId(sid int64) int64
	GetTimestamp(sid int64) int64
}

type SnowflakeSecond struct {
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

const secondStartEpoch = 1735660800

func NewSnowflakeSecond(workerId, workerIdBits, sequenceBits int64) *SnowflakeSecond {
	workeridMax := int64(-1 ^ (-1 << workerIdBits))
	if workerId < 0 || workerId > workeridMax {
		panic(fmt.Sprintf("workerid must be between 0 and %d", workeridMax))
	}
	sequenceMask := int64(-1 ^ (-1 << sequenceBits))
	timestampShift := workerIdBits + sequenceBits
	return &SnowflakeSecond{
		epoch:          secondStartEpoch,
		workeridBits:   workerIdBits,
		sequenceBits:   sequenceBits,
		sequenceMask:   sequenceMask,
		timestampShift: timestampShift,
		workerid:       workerId,
		sequence:       0,
	}
}

func (s *SnowflakeSecond) getRelativeTimestamp() int64 {
	return time.Now().Unix() - s.epoch
}

func (s *SnowflakeSecond) Next() int64 {
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
		s.sequence = 0
	} else {
		// 不同时间戳（精度：秒）下直接使用序列号：0
		s.sequence = 0
	}

	s.relativeTimestamp = now
	return (now << s.timestampShift) | (s.workerid << s.sequenceBits) | s.sequence
}

func (s *SnowflakeSecond) GetWorkerId(sid int64) int64 {
	workeridMax := int64(-1 ^ (-1 << s.workeridBits))
	return (sid >> s.sequenceBits) & workeridMax
}

func (s *SnowflakeSecond) GetTimestamp(sid int64) int64 {
	timestampBits := 63 - s.workeridBits - s.sequenceBits
	timestampMax := int64(-1 ^ (-1 << timestampBits))
	return (sid>>(s.sequenceBits+s.workeridBits))&timestampMax + s.epoch
}

type SnowflakeMillis struct {
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

const millisecondStartEpoch = 1735660800000

func NewSnowflakeMillis(workerId, workerIdBits, sequenceBits int64) *SnowflakeMillis {
	workeridMax := int64(-1 ^ (-1 << workerIdBits))
	if workerId < 0 || workerId > workeridMax {
		panic(fmt.Sprintf("workerid must be between 0 and %d", workeridMax))
	}
	sequenceMask := int64(-1 ^ (-1 << sequenceBits))
	timestampShift := workerIdBits + sequenceBits
	return &SnowflakeMillis{
		epoch:          millisecondStartEpoch,
		workeridBits:   workerIdBits,
		sequenceBits:   sequenceBits,
		sequenceMask:   sequenceMask,
		timestampShift: timestampShift,
		workerid:       workerId,
		sequence:       0,
	}
}

// 默认构造函数
//
// timestamp 占 42 位：可表示 139 年
//
// workid 占 9 位：可表示 512 个节点；
//
// sequence占 12 位：一毫秒内可生成 4096 个 ID
func NewSnowflakeMillisDefault(workerId int64) *SnowflakeMillis {
	return NewSnowflakeMillis(workerId, 9, 12)
}

func (s *SnowflakeMillis) getRelativeTimestamp() int64 {
	return time.Now().UnixMilli() - s.epoch
}

func (s *SnowflakeMillis) Next() int64 {
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
		s.sequence = 0
	} else {
		// 不同时间戳（精度：秒）下直接使用序列号：0
		s.sequence = 0
	}

	s.relativeTimestamp = now
	return (now << s.timestampShift) | (s.workerid << s.sequenceBits) | s.sequence
}

func (s *SnowflakeMillis) GetWorkerId(sid int64) int64 {
	workeridMax := int64(-1 ^ (-1 << s.workeridBits))
	return (sid >> s.sequenceBits) & workeridMax
}

func (s *SnowflakeMillis) GetTimestamp(sid int64) int64 {
	timestampBits := 63 - s.workeridBits - s.sequenceBits
	timestampMax := int64(-1 ^ (-1 << timestampBits))
	return (sid>>(s.sequenceBits+s.workeridBits))&timestampMax + s.epoch
}
