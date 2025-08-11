package ids

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

func init() {
	uuid.EnableRandPool()

	_, err := rand.Read(seqIDProcessUnique[:])
	if err != nil {
		panic(fmt.Errorf("cannot initialize SeqID package with crypto.rand.Reader: %w", err))
	}

	var b [4]byte
	_, err = rand.Read(b[:])
	if err != nil {
		panic(fmt.Errorf("cannot initialize SeqID package with crypto.rand.Reader: %w", err))
	}

	seqIDCounter = (uint32(b[0]) << 0) | (uint32(b[1]) << 8) | (uint32(b[2]) << 16) | (uint32(b[3]) << 24)
}

// 生成没有短横线的UUID字符串: 使用 uuidv7
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

var (
	ErrInvalidID = errors.New("invalid ID")
)

// 相较于 UUIDv7，SeqID 更短
//
// 规则如下：前 4 字节为秒时间戳，中间 5 字节为机器 ID，最后 3 字节为 序列号(参考了 golang  ObjectID 的实现)
type SeqID [12]byte

var NilSeqID SeqID

var seqIDCounter uint32
var seqIDProcessUnique [5]byte
var _ encoding.TextMarshaler = (*SeqID)(nil)
var _ encoding.TextUnmarshaler = (*SeqID)(nil)
var _ json.Marshaler = (*SeqID)(nil)
var _ json.Unmarshaler = (*SeqID)(nil)

// 生成一个全局唯一 ID（精度秒: 复用 mongo-driver 中 golang ObjectID 实现）
func NewSeqID() SeqID {
	var b [12]byte
	binary.BigEndian.PutUint32(b[0:4], uint32(time.Now().Unix()))
	copy(b[4:9], seqIDProcessUnique[:])
	seq := atomic.AddUint32(&seqIDCounter, 1)
	b[9] = byte(seq >> 16)
	b[10] = byte(seq >> 8)
	b[11] = byte(seq)

	return b
}

// 从 16 进制字符串生成 SeqID
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
	v, ok := value.([]byte)
	if ok {
		if len(v) == 12 {
			copy(u[:], v)
			return nil
		}
	}
	if v, ok := value.(SeqID); ok {
		*u = v
		return nil
	}
	return nil
}

// 写入数据库时序列化
func (u SeqID) Value() (driver.Value, error) {
	return u[:], nil
}

func (id SeqID) Timestamp() time.Time {
	unixSecs := binary.BigEndian.Uint32(id[0:4])
	return time.Unix(int64(unixSecs), 0).UTC()
}

func (id SeqID) Base64() string {
	// 使用 base64.RawURLEncoding 编码，去掉 padding
	return base64.RawURLEncoding.EncodeToString(id[:])
}

func (id SeqID) String() string {
	return hex.EncodeToString(id[:])
}

// 如果所有字节都为 0，则为 NilSeqID
func (id SeqID) IsNil() bool {
	return id == NilSeqID
}

func (id SeqID) MarshalText() ([]byte, error) {
	var buf [24]byte
	hex.Encode(buf[:], id[:])
	return buf[:], nil
}

func (id *SeqID) UnmarshalText(b []byte) error {
	// NB(charlie): The json package will use UnmarshalText instead of
	// UnmarshalJSON if the value is a string.

	// An empty string is not a valid SeqID, but we treat it as a
	// special value that decodes as NilSeqID.
	if len(b) == 0 {
		return nil
	}

	*id = NewSeqIDFromHex(string(b))
	return nil
}

func (id SeqID) MarshalJSON() ([]byte, error) {
	var buf [26]byte
	buf[0] = '"'
	hex.Encode(buf[1:25], id[:])
	buf[25] = '"'
	return buf[:], nil
}

func (id *SeqID) UnmarshalJSON(b []byte) error {
	// Ignore "null" to keep parity with the standard library. Decoding a JSON
	// null into a non-pointer SeqID field will leave the field unchanged.
	// For pointer values, encoding/json will set the pointer to nil and will
	// not enter the UnmarshalJSON hook.
	if string(b) == "null" {
		return nil
	}

	// Handle string
	if len(b) >= 2 && b[0] == '"' {
		return id.UnmarshalText(b[1 : len(b)-1])
	}
	if len(b) == 12 {
		copy(id[:], b)
		return nil
	}

	return ErrInvalidID
}
