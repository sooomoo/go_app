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

var nodeId int64

const (
	// timestamp(ms): 43bit
	nodeIDBits     = 8  // 最多 256 节点
	counterBits    = 12 // 一个节点每毫秒可生成 4096 个 ID
	timestampShift = nodeIDBits + counterBits
	maxSequence    = int64(-1 ^ (-1 << counterBits))
	bigIDEpoch     = 1735660800000
	bigIDMin       = 15980274272700000 // 生成的 ID 不应该小于此值
	bigIDMinLen    = 17
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
	nodeIdMax := int64(-1 ^ (-1 << nodeIDBits))
	if id < 0 || id >= nodeIdMax {
		panic(fmt.Sprintf("workerid must be in range [0,%d)", nodeIdMax))
	}

	nodeId = id
}

type BigID int64

var NilBigID BigID

var bigIDTimestamp int64
var bigIDCounter int64
var bigMutex = sync.Mutex{}

func NewID() int64 {
	bigMutex.Lock()
	defer bigMutex.Unlock()

	now := time.Now().UnixMilli()
	if now == bigIDTimestamp {
		// 当同一时间戳（精度：毫秒）下多次生成id会增加序列号
		bigIDCounter++
		if bigIDCounter > maxSequence {
			// 当前序列 Id 已经使用完，则需要等待下一毫秒
			for now <= bigIDTimestamp {
				time.Sleep(time.Microsecond * 10)
				now = time.Now().UnixMilli()
			}
			bigIDCounter = 0
		}
	} else if now < bigIDTimestamp {
		// 时钟回拨：等待到下一个时间戳
		for now < bigIDTimestamp {
			time.Sleep(time.Microsecond * 10)
			now = time.Now().UnixMilli()
		}
		bigIDCounter = 0
	} else {
		// 不同时间戳（精度：毫秒）下直接使用序列号：0
		bigIDCounter = 0
	}

	bigIDTimestamp = now
	return ((now - bigIDEpoch) << timestampShift) | (nodeId << counterBits) | bigIDCounter
}

func NewBigID() BigID {
	return BigID(NewID())
}

func NewBigIDFromString(str string) BigID {
	if len(str) < bigIDMinLen {
		return NilBigID
	}
	v, err := strconv.ParseInt(str, 10, 64)
	if err != nil || v < bigIDMin {
		return NilBigID
	}
	return BigID(v)
}

func (id BigID) ToInt64() int64 {
	return int64(id)
}

func (id BigID) Timestamp() time.Time {
	timestampBits := 63 - nodeIDBits - counterBits
	timestampMax := int64(-1 ^ (-1 << timestampBits))
	ms := (int64(id)>>(counterBits+nodeIDBits))&timestampMax + bigIDEpoch
	return time.UnixMilli(ms).UTC()
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
