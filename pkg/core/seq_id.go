package core

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

var (
	ErrInvalidHex = errors.New("hex string is not a valid SeqID")
	ErrInvalidID  = errors.New("invalid ID")
)

var processUnique [3]byte

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

type SeqID [10]byte

var NilSeqID SeqID

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

// IsZero returns true if id is the empty SeqID.
func (id SeqID) IsZero() bool {
	return id == NilSeqID
}

// NewSeqIDFromHex creates a SeqID from a hex string.
func NewSeqIDFromHex(s string) (SeqID, error) {
	if len(s) != 20 {
		return NilSeqID, ErrInvalidHex
	}

	var oid SeqID
	_, err := hex.Decode(oid[:], []byte(s))
	if err != nil {
		return NilSeqID, err
	}

	return oid, nil
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
	oid, err := NewSeqIDFromHex(string(b))
	if err != nil {
		return err
	}
	*id = oid
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
