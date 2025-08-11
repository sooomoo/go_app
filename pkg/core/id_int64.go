package core

import (
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var snowNodeId int64

const (
	// timestamp(ms): 42bit
	snowNodeIDBits     = 8  // 最多 256 节点
	snowCounterBits    = 12 // 一个节点每毫秒可生成 8192 个 ID
	snowTimestampShift = snowNodeIDBits + snowCounterBits
	snowMaxSequence    = int64(-1 ^ (-1 << snowCounterBits))
	snowIDEpoch        = int64(1735660800000) // 2025-01-01 00:00:00 UTC
	snowIDMin          = 40303604944347136    // 生成的 ID 不应该小于此值
	snowIDMinLen       = 17
)

func init() {
	idstr := strings.TrimSpace(os.Getenv("node_id"))
	if len(idstr) == 0 {
		panic("cannot find 'node_id' in env varibles")
	}
	id, err := strconv.ParseInt(idstr, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("'node_id' parse error: %s", err))
	}
	nodeIdMax := int64(-1 ^ (-1 << snowNodeIDBits))
	if id < 0 || id >= nodeIdMax {
		panic(fmt.Sprintf("workerid must be in range [0,%d)", nodeIdMax))
	}

	snowNodeId = id
}

var snowIDClockBackwardCallback func(time int64)

// 设置时钟回拨时的回调函数
func IDClockBackwardCallback(cb func(time int64)) {
	snowIDClockBackwardCallback = cb
}

var snowIDClockRestoreCallback func()

// 设置时钟恢复时的回调函数
func IDClockRestoreCallback(cb func()) {
	snowIDClockRestoreCallback = cb
}

// 在应用启动时设置，此方法主要防止以下情况：
// 在回拨过程中，服务器挂了。
// 这种情况好像也不用考虑，重启一般都需要几秒甚至更久，而回拨一般不会超过几百毫秒，所以影响不大
func IDClockBackwardPoint(timePoint int64) {
	snowIDMutex.Lock()
	defer snowIDMutex.Unlock()
	snowIDTimeBackPoint = timePoint
}

var snowIDSeq int64
var snowIDMutex sync.Mutex
var snowIDTimestamp int64
var snowIDTimeBackPoint int64

// var snowIDNowMillisFunc func() int64

// // TEST only
// func SetSnowIDNowMillisFunc(fn func() int64) {
// 	snowIDNowMillisFunc = fn
// }

// // TEST only：可以使用此函数模拟时钟回退
// func snowIDNowMillis() int64 {
// 	if snowIDNowMillisFunc != nil {
// 		return snowIDNowMillisFunc()
// 	}
// 	return time.Now().UnixMilli()
// }

// 生成一个全局唯一 ID (雪花算法的自定义实现，精度毫秒级)
//
// 规则: 63 位(第1位不用), 42位时间戳(2^42 毫秒可以表示 138 年), 8 位节点 ID, 13 位序列号
//
// nodeID: 8 位，即最多支持 256 个节点 (节点的值会在 init 函数自动从环境变量中获取， key 为 'node_id')
//
// time back: 1位，表示这个 ID 是在时钟回拨时生成的
//
// counter: 12 位，即每毫秒最多可生成 4096 个 ID
func NewID() int64 {
	snowIDMutex.Lock()
	defer snowIDMutex.Unlock()

	now := time.Now().UnixMilli()
	if now > snowIDTimeBackPoint { // 时钟回拨已经追赶上了，重置回拨时间点；或者没有产生回拨
		if snowIDTimeBackPoint > 0 {
			snowIDTimeBackPoint = 0
			if snowIDClockRestoreCallback != nil {
				snowIDClockRestoreCallback()
			}
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
				time.Sleep(time.Microsecond * 10)
				now = time.Now().UnixMilli()
				if now < snowIDTimestamp {
					// 产生了回拨
					if snowIDTimeBackPoint > 0 {
						// 回拨过程中，又产生了回拨，这种情况出现概率极低，直接 panic
						panic("core: unexpected time back occurred")
					}
					snowIDTimeBackPoint = snowIDTimestamp
					if snowIDClockBackwardCallback != nil {
						snowIDClockBackwardCallback(snowIDTimestamp)
					}
					break
				}
			}
		}
	} else if now > snowIDTimestamp { // 下一个时间戳了，序列号需要从0开始
		snowIDSeq = 0
	} else { // 时钟回拨
		if snowIDTimeBackPoint > 0 {
			// 回拨过程中，又产生了回拨，这种情况出现概率极低，直接 panic
			panic("core: unexpected time back occurred")
		}
		snowIDTimeBackPoint = snowIDTimestamp
		if snowIDClockBackwardCallback != nil {
			snowIDClockBackwardCallback(snowIDTimestamp)
		}
		// 不同时间戳（精度：毫秒）下直接使用序列号：0
		snowIDSeq = 0
	}

	snowIDTimestamp = now
	if snowIDTimeBackPoint > 0 {
		// 时钟有回拨
		return ((now - snowIDEpoch) << 21) | (snowNodeId << 13) | int64(1)<<12 | int64(snowIDSeq)
	} else {
		// 没有时钟回拨
		return ((now - snowIDEpoch) << 21) | (snowNodeId << 13) | int64(0)<<12 | int64(snowIDSeq)
	}
}

// 获取 NewID 生成的 ID 的时间戳
func IDTimestamp(snowId int64) time.Time {
	sec := snowId >> 21
	return time.UnixMilli(sec + snowIDEpoch).UTC()
}

// 获取 NewID 生成的 ID 的节点 ID
func IDNodeID(snowId int64) int64 {
	return (snowId >> 13) & 0xF
}

func IDTimeIsBack(snowId int64) bool {
	return (snowId>>12)&0b1 > 0
}

// 支持自定义序列化的 int64 ID
// 用于支持需要将 ID 序列化为字符串的场景
type BigID int64

var NilBigID BigID
var _ encoding.TextMarshaler = (*BigID)(nil)
var _ encoding.TextUnmarshaler = (*BigID)(nil)
var _ json.Marshaler = (*BigID)(nil)
var _ json.Unmarshaler = (*BigID)(nil)

func NewBigID() BigID {
	return BigID(NewID())
}

func NewBigIDFromString(str string) BigID {
	if len(str) < snowIDMinLen {
		return NilBigID
	}
	v, err := strconv.ParseInt(str, 10, 64)
	if err != nil || v < snowIDMin {
		return NilBigID
	}
	return BigID(v)
}

func (id BigID) ToInt64() int64 {
	return int64(id)
}

func (id BigID) Timestamp() time.Time {
	return IDTimestamp(int64(id))
}

func (id BigID) NodeID() int64 {
	return IDNodeID(int64(id))
}

func (id BigID) TimeIsBack() bool {
	return IDTimeIsBack(int64(id))
}

func (id BigID) String() string {
	return fmt.Sprintf("BigID(%d)", id)
}

// 从数据库读取时反序列化
func (u *BigID) Scan(value any) error {
	*u, _ = value.(BigID)
	return nil
}

// 写入数据库时序列化
func (id BigID) Value() (driver.Value, error) {
	return int64(id), nil
}

func (id BigID) IsNil() bool {
	return id == NilBigID
}

func (id BigID) MarshalText() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(id), 10)), nil
}

func (id *BigID) UnmarshalText(b []byte) error {
	// NB(charlie): The json package will use UnmarshalText instead of
	// UnmarshalJSON if the value is a string.

	// An empty string is not a valid ObjectID, but we treat it as a
	// special value that decodes as NilObjectID.
	if len(b) == 0 {
		return nil
	}
	oid := NewBigIDFromString(string(b))
	*id = oid
	return nil
}

func (id BigID) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, `"%d"`, id), nil
}

func (id *BigID) UnmarshalJSON(b []byte) error {
	// Ignore "null" to keep parity with the standard library. Decoding a JSON
	// null into a non-pointer ObjectID field will leave the field unchanged.
	// For pointer values, encoding/json will set the pointer to nil and will
	// not enter the UnmarshalJSON hook.
	if string(b) == "null" {
		return nil
	}

	// Handle string
	if len(b) >= 2 && b[0] == '"' {
		// TODO: fails because of error
		return id.UnmarshalText(b[1 : len(b)-1])
	}

	return ErrInvalidID
}
