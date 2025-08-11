package ids

import (
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

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
	return IDGetTimestamp(int64(id))
}

func (id BigID) NodeID() int64 {
	return IDGetNodeID(int64(id))
}

func (id BigID) TimeIsBack() bool {
	return IDHasClockBackward(int64(id))
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
