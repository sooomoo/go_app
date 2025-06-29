package core

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	if _, err := io.ReadFull(rand.Reader, processUnique[:]); err != nil {
		panic("failed to generate process unique identifier: " + err.Error())
	}
	// 程序启动时初始化 seqIDCounter
	// 使用 crypto/rand.Reader 生成一个随机数作为初始计数器值
	// 防止每次启动时计数器从 0 开始
	var b [3]byte
	_, err := io.ReadFull(rand.Reader, b[:])
	if err != nil {
		panic(fmt.Errorf("cannot initialize objectid package with crypto.rand.Reader: %w", err))
	}
	seqIDCounter = (uint32(b[0]) << 0) | (uint32(b[1]) << 8) | (uint32(b[2]) << 16)
}

// 相较于 UUIDv7/v8，SeqID 更加具有顺序性，每秒内有一个 counter 用于表示当前秒内的序列号
//
// 规则如下：前 4 字节为时间戳，中间 3 字节为进程唯一标识（随机生成），最后 3 字节为序列号(参考了 golang ObjectID 的实现)
type SeqID [10]byte

var NilSeqID SeqID

var processUnique [3]byte
var seqIDCounter uint32 = 0

// 生成一个全局唯一 ID (SeqID 自定义实现，精度秒级)
func NewSeqID() SeqID {
	// 不用担心时钟回拨，因为 seqIDCounter 的表示范围为 2^24，所以最多每秒产生 16777216 个 ID，
	// 就算时钟回拨，ID 的增量足以应对
	var b [10]byte
	now := time.Now().Unix()
	binary.BigEndian.PutUint32(b[0:4], uint32(now))

	// 测试时使用，模拟多个进程同时启动
	// var process [3]byte
	// // 初始化进程唯一标识符
	// if _, err := io.ReadFull(rand.Reader, process[:]); err != nil {
	// 	panic("failed to generate process unique identifier: " + err.Error())
	// }
	// copy(b[4:7], process[:])
	copy(b[4:7], processUnique[:])
	seq := atomic.AddUint32(&seqIDCounter, 1)
	seq &= 0x00FFFFFF
	// seq 取低24位，不用担心 snowIDSeq 超出 0x00FFFFFF 的情况
	// 因为当它超出 0x00FFFFFF 时，会自动回绕到 0x00000000
	// 且此时早已不在之前的时间戳内了
	b[7] = byte(seq >> 16)
	b[8] = byte(seq >> 8)
	b[9] = byte(seq)

	return b
}

// NewSeqIDFromHex creates a SeqID from a hex string.
func NewSeqIDFromHex(s string) SeqID {
	if len(s) != 20 {
		return NilSeqID
	}

	var oid SeqID
	_, err := hex.Decode(oid[:], []byte(s))
	if err != nil {
		return NilSeqID
	}

	return oid
}

func (id SeqID) Timestamp() time.Time {
	unixSecs := binary.BigEndian.Uint32(id[0:4])
	return time.Unix(int64(unixSecs), 0).UTC()
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
