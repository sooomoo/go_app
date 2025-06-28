package core

import (
	"database/sql/driver"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var snowNodeId int64

const (
	// timestamp(ms): 43bit
	snowNodeIDBits     = 8  // 最多 256 节点
	snowCounterBits    = 12 // 一个节点每毫秒可生成 4096 个 ID
	snowTimestampShift = snowNodeIDBits + snowCounterBits
	snowMaxSequence    = int64(-1 ^ (-1 << snowCounterBits))
	snowIDEpoch        = 1735660800000
	snowIDMin          = 15980274272700000 // 生成的 ID 不应该小于此值
	snowIDMinLen       = 17
)

func init() {
	idstr := os.Getenv("node_id")
	if len(idstr) == 0 {
		if strings.EqualFold(os.Getenv("env"), "prod") {
			panic("cannot find 'node_id' in env varibles")
		}
		idstr = "1"
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

var snowIDTimestamp int64
var snowIDCounter int64
var snowIDMutex = sync.Mutex{}

// 生成一个全局唯一 ID (雪花算法的自定义实现)
//
// nodeID: 8 位，即最多支持 256 个节点 (节点的值会在 init 函数自动从环境变量中获取， key 为 'node_id')
//
// counter: 12 位，即每毫秒最多可生成 4096 个 ID
func NewID() int64 {
	snowIDMutex.Lock()
	defer snowIDMutex.Unlock()

	now := time.Now().UnixMilli()
	if now == snowIDTimestamp {
		// 当同一时间戳（精度：毫秒）下多次生成id会增加序列号
		snowIDCounter++
		if snowIDCounter > snowMaxSequence {
			// 当前序列 Id 已经使用完，则需要等待下一毫秒
			for now <= snowIDTimestamp {
				time.Sleep(time.Microsecond * 10)
				now = time.Now().UnixMilli()
			}
			snowIDCounter = 0
		}
	} else if now < snowIDTimestamp {
		// 时钟回拨：等待到下一个时间戳
		for now < snowIDTimestamp {
			time.Sleep(time.Microsecond * 10)
			now = time.Now().UnixMilli()
		}
		snowIDCounter = 0
	} else {
		// 不同时间戳（精度：毫秒）下直接使用序列号：0
		snowIDCounter = 0
	}

	snowIDTimestamp = now
	return ((now - snowIDEpoch) << snowTimestampShift) | (snowNodeId << snowCounterBits) | snowIDCounter
}

// 获取 NewID 生成的 ID 的时间戳
func SnowIDTimestamp(id int64) time.Time {
	timestampBits := 63 - snowNodeIDBits - snowCounterBits
	timestampMax := int64(-1 ^ (-1 << timestampBits))
	ms := (id>>(snowCounterBits+snowNodeIDBits))&timestampMax + snowIDEpoch
	return time.UnixMilli(ms).UTC()
}

// 支持自定义序列化的 int64 ID
// 用于支持需要将 ID 序列化为字符串的场景
type BigID int64

var NilBigID BigID

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
	return SnowIDTimestamp(id.ToInt64())
}

func (id BigID) String() string {
	return fmt.Sprintf("BigID(%d)", id)
}

func (id BigID) Value() (driver.Value, error) {
	return int64(id), nil
}

func (id BigID) IsZero() bool {
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
