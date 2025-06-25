package core

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidHex   = errors.New("hex string is not a valid SeqID")
	ErrInvalidSeqID = errors.New("invalid SeqID")
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
	cnt := (uint32(b[0]) << 0) | (uint32(b[1]) << 8) | (uint32(b[2]) << 16)
	seqIDCounter.Store(cnt)
}

type SeqID [10]byte

var NilSeqID SeqID

var seqIDCounter = atomic.Uint32{}

func NewSeqID() SeqID {
	var b [10]byte

	binary.BigEndian.PutUint32(b[0:4], uint32(time.Now().Unix()))
	copy(b[4:7], processUnique[:])
	var newV uint32
	for {
		old := seqIDCounter.Load() // 原子读取当前值
		newV = old + 1
		// 超过阈值 0x00FFFFFF (16,777,215) 则重置为 0
		if old >= 0x00FFFFFF {
			newV = 1
		}
		// CAS 原子更新：若当前值仍为 old，则更新为 new
		if seqIDCounter.CompareAndSwap(old, newV) {
			break // 更新成功则退出循环
		}
		// 更新失败说明其他协程已修改值，重试
	}

	// 取低24位
	b[7] = byte(newV >> 16)
	b[8] = byte(newV >> 8)
	b[9] = byte(newV)

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

	return ErrInvalidSeqID
}
