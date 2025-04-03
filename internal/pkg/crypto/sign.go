package crypto

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"sort"

	"github.com/sooomo/niu"
)

var bufferPool = niu.NewByteBufferPool(0, 1024)
var base64Encoding = base64.StdEncoding

const (
	SignatureLength = 64 // Ed25519签名长度
)

// 用于生成待签名的内容
func StringfyMap(params map[string]string) []byte {
	// 对参数名进行排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 拼接参数
	buf := bufferPool.Get()
	defer bufferPool.Put(buf)
	for _, k := range keys {
		buf.WriteString(fmt.Sprintf("%s=%s&", k, params[k]))
	}
	buf.Truncate(buf.Len() - 1) // 去掉最后一个&
	return buf.Bytes()
}

// 生成签名密钥对
//
// 公钥用于验证签名，私钥用于签名
// 返回的是经过base64编码的字符串
func NewSignKeyPair() (pubKey, priKey string, err error) {
	pub, pri, err := ed25519.GenerateKey(nil)
	if err != nil {
		return "", "", err
	}

	return base64Encoding.EncodeToString(pub), base64Encoding.EncodeToString(pri), nil
}

// 使用Ed25519签名
//
// priKey: 私钥，经过base64编码的字符串
// data: 待签名的数据
func Sign(priKey string, data []byte) (string, error) {
	pri, err := base64Encoding.DecodeString(priKey)
	if err != nil {
		return "", err
	}
	signature := ed25519.Sign(pri, data)
	return base64Encoding.EncodeToString(signature), nil
}

// 使用Ed25519验证签名
//
// pubKey: 公钥，经过base64编码的字符串
// data: 待验证的数据
func VerifySign(pubKey string, data []byte, signature string) (bool, error) {
	pub, err := base64Encoding.DecodeString(pubKey)
	if err != nil {
		return false, err
	}

	sig, err := base64Encoding.DecodeString(signature)
	if err != nil {
		return false, err
	}

	return ed25519.Verify(pub, data, sig), nil
}
