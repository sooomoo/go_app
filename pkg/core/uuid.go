package core

import (
	"strings"

	"github.com/google/uuid"
)

func init() {
	uuid.EnableRandPool()
}

// 生成UUID字符串
func NewUUID() string {
	val, err := uuid.NewV7()
	if err != nil {
		return ""
	}
	return val.String()
}

// 是否是合法的 UUID
func IsUUIDValid(s string) bool {
	err := uuid.Validate(s)
	return err == nil
}

// 生成没有短横线的UUID字符串
func NewUUIDWithoutDash() string {
	uid := NewUUID()
	idStr := strings.ReplaceAll(uid, "-", "")
	return idStr
}
