package crypto

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"goapp/internal/app"
	"goapp/pkg/core"
	"sort"
)

var bufferPool = core.NewByteBufferPool(0, 1024)
var base64Encoding = base64.RawURLEncoding // 不需要 padding，即不包含=字符

const (
	SignatureLength = 64 // Ed25519签名长度
)

func Base64Encode(data []byte) string {
	return base64Encoding.EncodeToString(data)
}
func Base64Decode(data string) ([]byte, error) {
	return base64Encoding.DecodeString(data)
}

// 用于生成待签名的内容
func StringfyMap(params map[string]string) []byte {
	if len(params) == 0 {
		return nil
	}

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

// 使用Ed25519签名
//
// priKey: 私钥
// data: 待签名的数据
func SignMap(mp map[string]string) (string, error) {
	key := app.GetGlobal().GetAuthConfig().SignKeyPair.PrivateKey
	pKey := app.GetGlobal().GetAuthConfig().SignKeyPair.PublicKey
	if len(key) == 0 {
		panic("sign key is empty")
	}
	priKey, err := base64Encoding.DecodeString(key)
	if err != nil {
		return "", err
	}
	pubKey, err := base64Encoding.DecodeString(pKey)
	if err != nil {
		return "", err
	}

	data := StringfyMap(mp)
	fmt.Printf("data to sign: %s\n", string(data))
	priKey = append(priKey, pubKey...)
	signature := ed25519.Sign(priKey, data)
	return base64Encoding.EncodeToString(signature), nil
}

// 使用Ed25519验证签名
//
// pubKey: 公钥
// data: 待验证的数据
func VerifySignMap(pubKey []byte, mp map[string]string, signature string) (bool, error) {
	sig, err := base64Encoding.DecodeString(signature)
	if err != nil {
		return false, err
	}

	data := StringfyMap(mp)
	fmt.Printf("data to verify: %s\n", string(data))
	return ed25519.Verify(pubKey, data, sig), nil
}
