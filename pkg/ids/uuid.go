package ids

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInvalidID = errors.New("invalid id")
)

// 生成没有短横线的UUID字符串: 使用 uuidv7
func NewUUID() string {
	val, err := uuid.NewV7()
	if err != nil {
		return ""
	}
	uid := val.String()
	return strings.ReplaceAll(uid, "-", "")
}
