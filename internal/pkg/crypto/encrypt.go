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
func NegotiateShareKey(remotePubKey []byte, selfPriKeyBase64 string) ([]byte, error) {
	selfPriKey, err := base64Encoding.DecodeString(selfPriKeyBase64)
	if err != nil {
		return nil, err
	}
	return curve25519.X25519(selfPriKey, remotePubKey)
}

// Encrypt AES-GCM 加密
//
// key: 共享密钥
func Encrypt(secret []byte, data []byte) (string, error) {
	if len(data) == 0 {
		return "", nil
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce, err := niu.SecureBytes(aesgcm.NonceSize())
	if err != nil {
		return "", err
	}

	encrypted := aesgcm.Seal(nonce, nonce, data, nil)
	return base64Encoding.EncodeToString(encrypted), nil
}

// Decrypt AES-GCM 解密
//
// key: 共享密钥
func Decrypt(secret []byte, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	raw, err := base64Encoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}
	nonce := raw[:12]
	cipherData := raw[12:]

	res, err := aesgcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return nil, err
	}
	return res, nil
}
