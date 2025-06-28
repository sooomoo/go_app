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
	"sync"
	"time"

	"github.com/google/uuid"
)

func init() {
	uuid.EnableRandPool()

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

type UUIDv8 [10]byte

var NilUUIDv8 UUIDv8

const uuidv8StartEpochMs = 1735660800000

// 生成一个优化的UUID v8，10字节
func NewUUIDv8() UUIDv8 {
	// 获取当前时间戳（毫秒）
	now := uint64(time.Now().UnixMilli() - uuidv8StartEpochMs)

	// 构建UUID各部分
	uuid := UUIDv8{}

	// 6字节: 毫秒时间戳 (48位)
	binary.BigEndian.PutUint64(uuid[0:8], now<<16) // 高48位为时间戳
	// 4字节: 随机数部分 (32位)
	_, err := rand.Read(uuid[6:10])
	if err != nil {
		return NilUUIDv8
	}

	return uuid
}

func NewUUIDv8FromHex(str string) UUIDv8 {
	if len(str) != 20 {
		return NilUUIDv8
	}

	var oid UUIDv8
	_, err := hex.Decode(oid[:], []byte(str))
	if err != nil {
		return NilUUIDv8
	}

	return oid
}

// 是否是空UUID，即所有字节为 0
func (u UUIDv8) IsNil() bool {
	return u == NilUUIDv8
}

// ToString 将UUID字节切片转换为标准字符串格式
func (u UUIDv8) String() string {
	return `UUIDv8("` + u.Hex() + `")`
}

func (id UUIDv8) Timestamp() time.Time {
	unixSecs := binary.BigEndian.Uint64(id[0:8])
	unixSecs >>= 16
	return time.UnixMilli(int64(unixSecs)).UTC()
}

func (id UUIDv8) Hex() string {
	return hex.EncodeToString(id[:])
}

func (id UUIDv8) Base64() string {
	// 使用 base64.RawURLEncoding 编码，去掉 padding
	return base64.RawURLEncoding.EncodeToString(id[:])
}

func (id UUIDv8) MarshalText() ([]byte, error) {
	var buf [20]byte
	hex.Encode(buf[:], id[:])
	return buf[:], nil
}

func (id *UUIDv8) UnmarshalText(b []byte) error {
	// NB(charlie): The json package will use UnmarshalText instead of
	// UnmarshalJSON if the value is a string.

	// An empty string is not a valid ObjectID, but we treat it as a
	// special value that decodes as NilObjectID.
	if len(b) == 0 {
		return nil
	}
	*id = NewUUIDv8FromHex(string(b))
	return nil
}

func (id UUIDv8) MarshalJSON() ([]byte, error) {
	var buf [22]byte
	buf[0] = '"'
	hex.Encode(buf[1:21], id[:])
	buf[21] = '"'
	return buf[:], nil
}

func (id *UUIDv8) UnmarshalJSON(b []byte) error {
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

var (
	ErrInvalidHex = errors.New("hex string is not a valid SeqID")
	ErrInvalidID  = errors.New("invalid ID")
)

type SeqID [10]byte

var NilSeqID SeqID

var processUnique [3]byte
var seqIDLastTimestamp int64 = 0
var seqIDCounter uint32 = 0
var seqIDMutex = sync.Mutex{}

func NewSeqID() SeqID {
	seqIDMutex.Lock()
	defer seqIDMutex.Unlock()

	var b [10]byte
	now := time.Now().Unix()
	if now == seqIDLastTimestamp {
		seqIDCounter++
		// 超过阈值 0x00FFFFFF (16,777,215) 则重置为 0
		if seqIDCounter >= 0x00FFFFFF {
			for now <= seqIDLastTimestamp {
				time.Sleep(time.Microsecond * 10)
				now = time.Now().Unix()
			}
			seqIDCounter = 0
		}
	} else if now < seqIDLastTimestamp {
		// 时钟回拨：等待到下一个时间戳
		for now < seqIDLastTimestamp {
			time.Sleep(time.Microsecond * 10)
			now = time.Now().Unix()
		}
		seqIDCounter = 0
	} else {
		// 不同时间戳（精度：秒）下直接使用序列号：0
		seqIDCounter = 0
	}

	seqIDLastTimestamp = now

	binary.BigEndian.PutUint32(b[0:4], uint32(now))
	copy(b[4:7], processUnique[:])

	// 取低24位
	b[7] = byte(seqIDCounter >> 16)
	b[8] = byte(seqIDCounter >> 8)
	b[9] = byte(seqIDCounter)

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
