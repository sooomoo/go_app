package ids

import (
	"database/sql/driver"
	"encoding"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type UID [16]byte

var ZeroUID UID

var _ encoding.TextMarshaler = (*UID)(nil)
var _ encoding.TextUnmarshaler = (*UID)(nil)
var _ json.Marshaler = (*UID)(nil)
var _ json.Unmarshaler = (*UID)(nil)

var uidFailCallback func(err error)

// 设置一个 uid 生成失败时的回调函数
func SetUIDFailCallback(f func(err error)) {
	uidFailCallback = f
}

// 生成一个全局唯一 ID, 使用 uuidv7 生成
func NewUID() UID {
	uuid, err := uuid.NewV7()
	if err != nil {
		if uidFailCallback != nil {
			uidFailCallback(fmt.Errorf("failed to generate UID: %w", err))
		} else {
			log.Printf("failed to generate UID: %v", err)
		}
		return ZeroUID
	}
	return UID(uuid)
}

// 从 16 进制字符串生成 UID
func NewUIDFromHex(s string) (UID, error) {
	var oid UID
	id, err := uuid.ParseBytes([]byte(s))
	if err != nil {
		return oid, err
	}
	if id.Version() != uuid.Version(7) {
		return oid, fmt.Errorf("invalid UUID version (expected %d, got %d)", uuid.Version(7), id.Version())
	}
	return UID(id), nil
}

// 从 base64 字符串生成 UID
func NewUIDFromBase64(s string) (UID, error) {
	out, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		var oid UID
		return oid, err
	}
	if len(out) != 16 {
		var oid UID
		return oid, fmt.Errorf("invalid UUID length (expected 16 bytes, got %d bytes)", len(out))
	}
	return UID(out), nil
}

func (u UID) ToBase64() string {
	// 使用 base64.RawURLEncoding 编码，去掉 padding
	return base64.RawURLEncoding.EncodeToString(u[:])
}

func (u UID) String() string {
	return hex.EncodeToString(u[:])
}

// 如果所有字节都为 0，则为 ZeroUID
func (id UID) IsZero() bool {
	// 由于 [16]byte是固定长度的数组（非切片），其类型本身支持 ==操作符。比较时会逐字节检查每个元素的值​：
	return id == ZeroUID
}

// 从数据库读取时反序列化
func (u *UID) Scan(value any) error {
	switch src := value.(type) {
	case nil:
		return nil

	case string:
		// if an empty UUID comes from a table, we return a null UUID
		if src == "" {
			return nil
		}

		// see Parse for required string format
		uid, err := uuid.Parse(src)
		if err != nil {
			return fmt.Errorf("Scan: %v", err)
		}

		*u = UID(uid)
	case []byte:
		// if an empty UUID comes from a table, we return a null UUID
		if len(src) == 0 {
			return nil
		}

		// assumes a simple slice of bytes if 16 bytes
		// otherwise attempts to parse
		if len(src) != 16 {
			return u.Scan(string(src))
		} else {
			copy((*u)[:], src)
		}
	default:
		return fmt.Errorf("Scan: unable to scan type %T into UUID", src)
	}

	return nil
}

// 写入数据库时序列化
func (u UID) Value() (driver.Value, error) {
	return u.String(), nil
}

// MarshalText implements encoding.TextMarshaler.
func (u UID) MarshalText() ([]byte, error) {
	val := hex.EncodeToString(u[:])
	return []byte(val), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (u *UID) UnmarshalText(data []byte) error {
	id, err := uuid.ParseBytes(data)
	if err != nil {
		return err
	}
	*u = UID(id)
	return nil
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (u UID) MarshalBinary() ([]byte, error) {
	return u[:], nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (u *UID) UnmarshalBinary(data []byte) error {
	if len(data) != 16 {
		return fmt.Errorf("invalid UID (got %d bytes)", len(data))
	}
	copy(u[:], data)
	return nil
}

func (id UID) MarshalJSON() ([]byte, error) {
	var buf [34]byte
	buf[0] = '"'
	hex.Encode(buf[1:33], id[:])
	buf[33] = '"'
	return buf[:], nil
}

func (id *UID) UnmarshalJSON(b []byte) error {
	// Ignore "null" to keep parity with the standard library. Decoding a JSON
	// null into a non-pointer SeqID field will leave the field unchanged.
	// For pointer values, encoding/json will set the pointer to nil and will
	// not enter the UnmarshalJSON hook.
	if string(b) == "null" || string(b) == "NULL" {
		return nil
	}

	// Handle string
	if len(b) >= 2 && b[0] == '"' {
		return id.UnmarshalText(b[1 : len(b)-1])
	}
	if len(b) == 16 {
		copy(id[:], b)
		if id.Version() != uuid.Version(7) {
			return fmt.Errorf("invalid UUID format")
		}
		return nil
	}

	return fmt.Errorf("invalid UID format")
}

// Variant returns the variant encoded in uuid.
func (id UID) Variant() uuid.Variant {
	switch {
	case (id[8] & 0xc0) == 0x80:
		return uuid.RFC4122
	case (id[8] & 0xe0) == 0xc0:
		return uuid.Microsoft
	case (id[8] & 0xe0) == 0xe0:
		return uuid.Future
	default:
		return uuid.Reserved
	}
}

// 此处应该为 7
func (id UID) Version() uuid.Version {
	return uuid.Version(id[6] >> 4)
}

func (id UID) ToUUID() uuid.UUID {
	var ret uuid.UUID
	copy(ret[:], id[:])
	return ret
}

// ID 生成的时间，单位是 milliseconds；如果无法解码返回 -1
func (id UID) TimeUnixMills() int64 {
	switch id.Version() {
	case 7:
		// data := []byte{0, 0}
		// data = append(data, id[:6]...)
		// time := binary.BigEndian.Uint64(data)
		time := int64(id[0])<<40 | int64(id[1])<<32 | int64(id[2])<<24 | int64(id[3])<<16 | int64(id[4])<<8 | int64(id[5])
		return time
	}
	return -1
}

// ID 生成的时间，单位是 seconds；如果无法解码返回 -1
func (id UID) TimeUnixSeconds() int64 {
	time := id.TimeUnixMills()
	if time == -1 {
		return time
	}
	return id.TimeUnixMills() / 1e3
}

func (id UID) Time() time.Time {
	return time.UnixMilli(id.TimeUnixMills())
}
