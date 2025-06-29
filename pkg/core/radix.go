package core

import (
	"fmt"
	"strings"
)

// 自定义34进制字符集（0-9, A-K, M-N, P-Z，共34个字符）
// 不包含 O、L 这两个容易混淆的字符
const customBase34Chars = "0123456789ABCDEFGHIJKMNPQRSTUVWXYZ"

// 自定义进制
type CustomRadix struct {
	chars string
}

func NewCustomRadix(chars string) *CustomRadix {
	return &CustomRadix{chars: chars}
}
func NewCustomRadix34() *CustomRadix {
	return &CustomRadix{chars: customBase34Chars}
}

// 将整数转换为自定义进制字符串
func (c *CustomRadix) Encode(num int) string {
	if num < 0 {
		return "-" + c.Encode(-num)
	}
	if num == 0 {
		return string(c.chars[0])
	}

	var result []byte
	base := len(c.chars)

	for num > 0 {
		remainder := num % base
		result = append([]byte{c.chars[remainder]}, result...)
		num /= base
	}

	return string(result)
}

// 将自定义进制字符串转换回整数
func (c *CustomRadix) Decode(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("输入不能为空")
	}

	// 处理负号
	neg := false
	if s[0] == '-' {
		if len(s) == 1 {
			return 0, fmt.Errorf("无效的负数格式")
		}
		neg = true
		s = s[1:]
	}

	base := len(c.chars)
	result := 0

	for _, char := range s {
		index := strings.IndexRune(c.chars, char)
		if index == -1 {
			return 0, fmt.Errorf("无效字符: %c", char)
		}
		result = result*base + index
	}

	if neg {
		result = -result
	}

	return result, nil
}
