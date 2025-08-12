package ids

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

var snowNodeId int64

const (
	snowNodeIDBits        = 7 // 最多 128 节点
	snowClockBackBits     = 2 // 使用 2 bit 标识时钟是否回拨
	snowClockBackMaxTimes = int64(-1 ^ (-1 << snowClockBackBits))
	snowCounterBits       = 13 // 一个节点每毫秒可生成 8192 个 ID
	snowTimestampShift    = snowNodeIDBits + snowCounterBits
	snowNodeIDMax         = int64(-1 ^ (-1 << snowNodeIDBits))
	snowMaxSequence       = int64(-1 ^ (-1 << snowCounterBits))
	snowIDEpoch           = int64(1735660800000) // 2025-01-01 00:00:00 UTC
	snowIDMin             = 40303604944347136    // 生成的 ID 不应该小于此值
	snowIDMinLen          = 17
)

// 从环境变量中初始化节点 ID
func IDSetNodeIDFromEnv(key string) error {
	idstr := strings.TrimSpace(os.Getenv(key))
	if len(idstr) == 0 {
		return fmt.Errorf("cannot find '%s' in env varibles", key)
	}
	id, err := strconv.ParseInt(idstr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse node ID from env var '%s': %v", key, err)
	}

	return IDSetNodeID(id)
}

// 设置节点ID
func IDSetNodeID(nodeID int64) error {
	if nodeID < 0 || nodeID >= snowNodeIDMax {
		return fmt.Errorf("nodeID must be in range [0,%d)", snowNodeIDMax)
	}
	snowIDMutex.Lock()
	defer snowIDMutex.Unlock()
	snowNodeId = nodeID
	return nil
}

var snowIDSeq int64
var snowIDMutex sync.Mutex
var snowIDTimestamp int64

// 最多可三次回拨：即回拨过程中，又回拨
// 第一个元素为第一次回拨
var snowIDClockBackPoints = [3]int64{0, 0, 0}

// 回退次数：最多三次
var snowIDClockBackTimes = int64(0)

var snowIDNowMillisFunc func() int64

// TEST only
func SetSnowIDNowMillisFunc(fn func() int64) {
	snowIDNowMillisFunc = fn
}

// 可以使用此函数模拟时钟回退
func snowIDNowMillis() int64 {
	if snowIDNowMillisFunc != nil {
		return snowIDNowMillisFunc()
	}
	return time.Now().UnixMilli()
}

// 生成一个全局唯一 ID (雪花算法的自定义实现，精度毫秒级)
//
// 规则: 63 位(第1位不用),
//
// timestamp: 41位, (2^41 毫秒可以表示 69 年)
//
// nodeID: 8 位，即最多支持 256 个节点 (节点的值会在 init 函数自动从环境变量中获取， key 为 'node_id')
//
// time back: 1位，表示这个 ID 是在时钟回拨时生成的
//
// counter: 13 位，即每毫秒最多可生成 8192 个 ID
//
// 以下情况可能不需要考虑：
// 在回拨过程中，服务器挂了：因为重启一般都需要几秒甚至更久，而回拨一般不会超过几百毫秒，所以影响不大
func NewID() int64 {
	snowIDMutex.Lock()
	defer snowIDMutex.Unlock()

	now := time.Now().UnixMilli()
	if now > snowIDClockBackPoints[0] {
		// 只有赶上第一次回拨，才能表示时钟已经恢复
		// 时钟回拨已经追赶上了，重置回拨时间点；或者没有产生回拨
		if snowIDClockBackPoints[0] > 0 {
			snowIDClockBackPoints[0] = 0
			snowIDClockBackPoints[1] = 0
			snowIDClockBackPoints[2] = 0
			snowIDClockBackTimes = 0
			log.Info().Msg("clock has been back to normal")
		}
	} else {
		// now == snowIDTimeBackPoint: 时间虽然已经追赶上，但还不能重置状态，因为回拨时，可能已经用了一些序列号了
		// now < snowIDTimeBackPoint: 仍然处于回拨状态
		now = snowIDTimestamp
	}

	if now == snowIDTimestamp {
		// 当同一时间戳（精度：毫秒）下多次生成id会增加序列号
		snowIDSeq = (snowIDSeq + 1) & snowMaxSequence
		if snowIDSeq == 0 {
			// 当前序列 Id 已经使用完，则需要等待下一秒
			for now <= snowIDTimestamp {
				time.Sleep(100 * time.Microsecond)
				now = time.Now().UnixMilli()
				if now < snowIDTimestamp {
					// 产生了回拨
					snowIDClockBackTimes = (snowIDClockBackTimes + 1)
					if snowIDClockBackTimes > 0 {
						log.Warn().Msgf("clock back happened at %d, new now time %d, times: %d", snowIDTimestamp, now, snowIDClockBackTimes)
					}
					if snowIDClockBackTimes > 3 {
						// 回拨过程中，又产生了回拨，这种情况出现概率极低，直接 panic
						log.Fatal().Msgf("unexpected clock back occurred when waiting for next millisecond. original back time: %d", snowIDTimestamp)
						panic("ids: unexpected time back occurred")
					}
					snowIDClockBackPoints[snowIDClockBackTimes-1] = snowIDTimestamp
					break
				}
			}
		}
	} else if now > snowIDTimestamp { // 下一个时间戳了，序列号需要从0开始
		snowIDSeq = 0
	} else { // 时钟回拨
		snowIDClockBackTimes = (snowIDClockBackTimes + 1)
		if snowIDClockBackTimes > 0 {
			log.Warn().Msgf("clock back happened at %d, new now time %d, times: %d", snowIDTimestamp, now, snowIDClockBackTimes)
		}
		if snowIDClockBackTimes > 3 {
			// 回拨过程中，又产生了回拨，这种情况出现概率极低，直接 panic
			log.Fatal().Msgf("unexpected clock back occurred when waiting for next millisecond. original back time: %d", snowIDTimestamp)
			panic("ids: unexpected time back occurred")
		}
		snowIDClockBackPoints[snowIDClockBackTimes-1] = snowIDTimestamp
		// log.Warn().Msgf("clock back happened at %d, new now time %d.\n", snowIDTimestamp, now)
		// 不同时间戳（精度：毫秒）下直接使用序列号：0
		snowIDSeq = 0
	}

	snowIDTimestamp = now
	nodeOffset := snowClockBackBits + snowCounterBits
	timeOffset := snowNodeIDBits + nodeOffset
	return ((now - snowIDEpoch) << timeOffset) | (snowNodeId << nodeOffset) | (snowIDClockBackTimes << snowCounterBits) | int64(snowIDSeq)
}

// 获取 NewID 生成的 ID 的时间戳
func IDGetTimestamp(snowId int64) time.Time {
	nodeOffset := snowClockBackBits + snowCounterBits
	timeOffset := snowNodeIDBits + nodeOffset
	sec := snowId >> timeOffset
	return time.UnixMilli(sec + snowIDEpoch).UTC()
}

// 获取 NewID 生成的 ID 的节点 ID
func IDGetNodeID(snowId int64) int64 {
	nodeOffset := snowClockBackBits + snowCounterBits
	return (snowId >> nodeOffset) & snowNodeIDMax
}

// 获取 NewID 生成的 ID 是否经历了时钟回拨
func IDGetClockBackwardTimes(snowId int64) int64 {
	return (snowId >> snowCounterBits) & snowClockBackMaxTimes
}
