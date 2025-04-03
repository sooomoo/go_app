package crypto

import (
	"crypto/aes"
	"crypto/cipher"

	"github.com/sooomo/niu"
	"golang.org/x/crypto/curve25519"
)

// 生成用于协商的密钥对
func NewNegotiateKeyPair() (pubKey, priKey []byte, err error) {
	private, err := niu.SecureBytes(32)
	if err != nil {
		return nil, nil, err
	}

	private[0] &= 248
	private[31] &= 127
	private[31] |= 64

	public, err := curve25519.X25519(private[:], curve25519.Basepoint)
	if err != nil {
		return nil, nil, err
	}

	return public, private, nil
}

// 通过与对方公钥协商，获得共享密钥
func NegotiateShareKey(remotePubKey, selfPriKey []byte) ([]byte, error) {
	return curve25519.X25519(selfPriKey, remotePubKey)
}

// Encrypt AES-GCM 加密
//
// key: 共享密钥，必须是base64编码的32位字符串
func Encrypt(key string, data []byte) ([]byte, error) {
	secret, err := base64Encoding.DecodeString(key)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce, err := niu.SecureBytes(aesgcm.NonceSize())
	if err != nil {
		return nil, err
	}

	return aesgcm.Seal(data[:0], nonce, data, nil), nil
}

// Decrypt AES-GCM 解密
//
// key: 共享密钥，必须是base64编码的32位字符串
func Decrypt(key string, data []byte) ([]byte, error) {
	secret, err := base64Encoding.DecodeString(key)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := data[:aesgcm.NonceSize()]
	cipherData := data[aesgcm.NonceSize():]

	return aesgcm.Open(nil, nonce, cipherData, nil)
}
