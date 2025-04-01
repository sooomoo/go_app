package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

// 使用HMAC-SHA256进行签名
func Sign(data string, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// 验证签名
func VerifySign(data, signature, key string) bool {
	expectedSignature := Sign(data, key)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// AES-GCM加密
func Encrypt(plaintext string, key string) (string, error) {
	// 确保密钥长度为32字节(256位)
	keyBytes := []byte(key)
	if len(keyBytes) != 32 {
		return "", errors.New("invalid key")
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	// 创建GCM模式
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 创建随机nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AES-GCM解密
func Decrypt(encryptedText string, key string) (string, error) {
	// 确保密钥长度为32字节(256位)
	keyBytes := []byte(key)
	if len(keyBytes) < 32 {
		paddedKey := make([]byte, 32)
		copy(paddedKey, keyBytes)
		keyBytes = paddedKey
	} else if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 获取nonce大小
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
