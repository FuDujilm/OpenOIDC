package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const encryptedSecretVersion = "v1"

type SecretCipher struct {
	aead cipher.AEAD
}

func NewSecretCipher(keyMaterial string) (*SecretCipher, error) {
	keyMaterial = strings.TrimSpace(keyMaterial)
	if keyMaterial == "" {
		return nil, fmt.Errorf("client secret encryption key required")
	}

	key := decodeSecretCipherKey(keyMaterial)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AES-GCM: %w", err)
	}
	return &SecretCipher{aead: aead}, nil
}

func decodeSecretCipherKey(keyMaterial string) []byte {
	if decoded, err := base64.StdEncoding.DecodeString(keyMaterial); err == nil && len(decoded) == 32 {
		return decoded
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(keyMaterial); err == nil && len(decoded) == 32 {
		return decoded
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(keyMaterial); err == nil && len(decoded) == 32 {
		return decoded
	}
	if len(keyMaterial) == 32 {
		return []byte(keyMaterial)
	}
	sum := sha256.Sum256([]byte(keyMaterial))
	return sum[:]
}

func (c *SecretCipher) Encrypt(plain string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("read random nonce: %w", err)
	}
	sealed := c.aead.Seal(nil, nonce, []byte(plain), []byte(encryptedSecretVersion))
	payload := append(nonce, sealed...)
	return encryptedSecretVersion + ":" + base64.RawURLEncoding.EncodeToString(payload), nil
}

func (c *SecretCipher) Decrypt(encrypted string) (string, error) {
	version, encoded, ok := strings.Cut(encrypted, ":")
	if !ok || version != encryptedSecretVersion {
		return "", fmt.Errorf("invalid encrypted secret format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode encrypted secret: %w", err)
	}
	nonceSize := c.aead.NonceSize()
	if len(payload) <= nonceSize {
		return "", fmt.Errorf("encrypted secret payload too short")
	}
	nonce := payload[:nonceSize]
	sealed := payload[nonceSize:]
	plain, err := c.aead.Open(nil, nonce, sealed, []byte(version))
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(plain), nil
}
