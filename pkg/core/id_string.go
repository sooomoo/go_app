package core

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

func init() {
	uuid.EnableRandPool()
}

// 生成没有短横线的UUID字符串
func NewUUID() string {
	val, err := uuid.NewV7()
	if err != nil {
		return ""
	}
	uid := val.String()
	return strings.ReplaceAll(uid, "-", "")
}

// 是否是合法的 UUID
func IsUUIDValid(s string) bool {
	err := uuid.Validate(s)
	return err == nil
}

// // UUIDv8 10字节: 用于生成具有顺序性的UUID，前 5字节为毫秒时间戳（可表示 34.8 年），后 5字节为随机数
// type UUIDv8 [10]byte
// var NilUUIDv8 UUIDv8
// const uuidv8StartEpochMs = 1735660800000 // 2025-01-01 00:00:00 UTC
// // 生成一个优化的UUID v8，10字节
// func NewUUIDv8() UUIDv8 {
// 	// 获取当前时间戳（毫秒）
// 	now := uint64(time.Now().UnixMilli() - uuidv8StartEpochMs)
// 	// 构建UUID各部分
// 	uuid := UUIDv8{}
// 	// 5字节: 毫秒时间戳 (40位)，可以表示 34.8 年，够用了
// 	binary.BigEndian.PutUint64(uuid[0:8], now<<24) // 高48位为时间戳
// 	// 5字节: 随机数部分 (40位)
// 	_, err := rand.Read(uuid[5:10])
// 	if err != nil {
// 		return NilUUIDv8
// 	}
// 	return uuid
// }
// func NewUUIDv8FromHex(str string) UUIDv8 {
// 	if len(str) != 20 {
// 		return NilUUIDv8
// 	}
// 	var oid UUIDv8
// 	_, err := hex.Decode(oid[:], []byte(str))
// 	if err != nil {
// 		return NilUUIDv8
// 	}
// 	return oid
// }
// // 是否是空UUID，即所有字节为 0
// func (u UUIDv8) IsNil() bool {
// 	return u == NilUUIDv8
// }
// // ToString 将UUID字节切片转换为标准字符串格式
// func (u UUIDv8) String() string {
// 	return `UUIDv8("` + u.Hex() + `")`
// }
// func (id UUIDv8) Timestamp() time.Time {
// 	unixSecs := binary.BigEndian.Uint64(id[0:8])
// 	unixSecs >>= 24
// 	return time.UnixMilli(int64(unixSecs) + uuidv8StartEpochMs).UTC()
// }
// func (id UUIDv8) Hex() string {
// 	return hex.EncodeToString(id[:])
// }
// func (id UUIDv8) Base64() string {
// 	// 使用 base64.RawURLEncoding 编码，去掉 padding
// 	return base64.RawURLEncoding.EncodeToString(id[:])
// }
// func (id UUIDv8) MarshalText() ([]byte, error) {
// 	var buf [20]byte
// 	hex.Encode(buf[:], id[:])
// 	return buf[:], nil
// }
// func (id *UUIDv8) UnmarshalText(b []byte) error {
// 	// NB(charlie): The json package will use UnmarshalText instead of
// 	// UnmarshalJSON if the value is a string.
// 	// An empty string is not a valid ObjectID, but we treat it as a
// 	// special value that decodes as NilObjectID.
// 	if len(b) == 0 {
// 		return nil
// 	}
// 	*id = NewUUIDv8FromHex(string(b))
// 	return nil
// }
// func (id UUIDv8) MarshalJSON() ([]byte, error) {
// 	var buf [22]byte
// 	buf[0] = '"'
// 	hex.Encode(buf[1:21], id[:])
// 	buf[21] = '"'
// 	return buf[:], nil
// }
// func (id *UUIDv8) UnmarshalJSON(b []byte) error {
// 	// Ignore "null" to keep parity with the standard library. Decoding a JSON
// 	// null into a non-pointer ObjectID field will leave the field unchanged.
// 	// For pointer values, encoding/json will set the pointer to nil and will
// 	// not enter the UnmarshalJSON hook.
// 	if string(b) == "null" {
// 		return nil
// 	}
// 	// Handle string
// 	if len(b) >= 2 && b[0] == '"' {
// 		// TODO: fails because of error
// 		return id.UnmarshalText(b[1 : len(b)-1])
// 	}
// 	if len(b) == 10 {
// 		copy(id[:], b)
// 		return nil
// 	}
// 	return ErrInvalidID
// }

var (
	ErrInvalidID = errors.New("invalid ID")
)

func init() {
	// 初始化进程唯一标识符
	if _, err := rand.Read(processUnique[:]); err != nil {
		panic("failed to generate process unique identifier: " + err.Error())
	}
}

// 相较于 UUIDv7/v8，SeqID 更加具有顺序性，每毫秒内有一个 counter 用于表示当前毫秒内的序列号
//
// 规则如下：前 5 字节为时间戳，中间 3 字节为进程唯一标识（随机生成），最后 4 字节为序列号(参考了 golang ObjectID 的实现)
type SeqID [12]byte

var NilSeqID SeqID

var processUnique [3]byte
var seqIDCounter uint32 = 0
var seqIDEpoch = int64(1735660800000)

// 生成一个全局唯一 ID (SeqID 自定义实现，精度秒级)
func NewSeqID() SeqID {
	var b [12]byte
	now := time.Now().UnixMilli() - seqIDEpoch
	binary.BigEndian.PutUint64(b[0:8], uint64(now<<24))
	copy(b[5:8], processUnique[:])
	seq := atomic.AddUint32(&seqIDCounter, 1)
	binary.BigEndian.PutUint32(b[8:12], uint32(seq))

	return b
}

// NewSeqIDFromHex creates a SeqID from a hex string.
func NewSeqIDFromHex(s string) SeqID {
	if len(s) != 24 {
		return NilSeqID
	}

	var oid SeqID
	_, err := hex.Decode(oid[:], []byte(s))
	if err != nil {
		return NilSeqID
	}

	return oid
}

// 从数据库读取时反序列化
func (u *SeqID) Scan(value any) error {
	*u, _ = value.(SeqID)
	return nil
}

// 写入数据库时序列化
func (u SeqID) Value() (driver.Value, error) {
	return [12]byte(u), nil
}

func (id SeqID) Timestamp() time.Time {
	unixSecs := binary.BigEndian.Uint64(id[0:8])
	return time.UnixMilli(int64(unixSecs>>24) + seqIDEpoch).UTC()
}

func (id SeqID) ProcessID() string {
	return hex.EncodeToString(id[5:8])
}

func (id SeqID) Hex() string {
	return hex.EncodeToString(id[:])
}

func (id SeqID) Base64() string {
	// 使用 base64.RawURLEncoding 编码，去掉 padding
	return base64.RawURLEncoding.EncodeToString(id[:])
}

func (id SeqID) String() string {
	return `SeqID("` + id.Hex() + `")`
}

// 如果所有字节都为 0，则为 NilSeqID
func (id SeqID) IsNil() bool {
	return id == NilSeqID
}

func (id SeqID) MarshalText() ([]byte, error) {
	var buf [20]byte
	hex.Encode(buf[:], id[:])
	return buf[:], nil
}

func (id *SeqID) UnmarshalText(b []byte) error {
	// NB(charlie): The json package will use UnmarshalText instead of
	// UnmarshalJSON if the value is a string.

	// An empty string is not a valid ObjectID, but we treat it as a
	// special value that decodes as NilObjectID.
	if len(b) == 0 {
		return nil
	}

	*id = NewSeqIDFromHex(string(b))
	return nil
}

func (id SeqID) MarshalJSON() ([]byte, error) {
	var buf [22]byte
	buf[0] = '"'
	hex.Encode(buf[1:21], id[:])
	buf[21] = '"'
	return buf[:], nil
}

func (id *SeqID) UnmarshalJSON(b []byte) error {
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
	if len(b) == 10 {
		copy(id[:], b)
		return nil
	}

	return ErrInvalidID
}
