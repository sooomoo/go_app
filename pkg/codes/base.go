package codes

import (
	"fmt"
	"math/big"
)

var ErrInvalidInput = fmt.Errorf("invalid input")

const base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// base: 2~62
func ToString(bytes []byte, base int) string {
	// 不能使用 big.NewInt(0).SetBytes(bytes).Text(base)
	// 首字母为 0时，有可能丢失首字母
	return big.NewInt(0).SetBytes(bytes).Text(base)
}

// base: 2~62
func ToBytes(src string, base int) ([]byte, error) {
	n, ok := big.NewInt(0).SetString(src, base)
	if !ok {
		return nil, ErrInvalidInput
	}
	return n.Bytes(), nil
}
