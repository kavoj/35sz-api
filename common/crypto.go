package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const paymentSecretPrefix = "paycfg:v1:"

var paymentSecretAAD = []byte("new-api/payment-config/v1")

func GenerateHMACWithKey(key []byte, data string) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func GenerateHMAC(data string) string {
	h := hmac.New(sha256.New, []byte(CryptoSecret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func Password2Hash(password string) (string, error) {
	passwordBytes := []byte(password)
	hashedPassword, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.DefaultCost)
	return string(hashedPassword), err
}

func ValidatePasswordAndHash(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func paymentEncryptionKey() [32]byte {
	return sha256.Sum256([]byte("payment-config-v1:" + CryptoSecret))
}

func EncryptPaymentSecret(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key := paymentEncryptionKey()
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", fmt.Errorf("initialize payment secret cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("initialize payment secret gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate payment secret nonce: %w", err)
	}
	sealed := gcm.Seal(nil, nonce, []byte(plaintext), paymentSecretAAD)
	payload := append(nonce, sealed...)
	return paymentSecretPrefix + base64.RawURLEncoding.EncodeToString(payload), nil
}

func DecryptPaymentSecret(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if !strings.HasPrefix(ciphertext, paymentSecretPrefix) {
		return "", errors.New("unsupported payment secret format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(ciphertext, paymentSecretPrefix))
	if err != nil {
		return "", errors.New("invalid payment secret encoding")
	}
	key := paymentEncryptionKey()
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", fmt.Errorf("initialize payment secret cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("initialize payment secret gcm: %w", err)
	}
	if len(payload) < gcm.NonceSize() {
		return "", errors.New("invalid payment secret payload")
	}
	plaintext, err := gcm.Open(nil, payload[:gcm.NonceSize()], payload[gcm.NonceSize():], paymentSecretAAD)
	if err != nil {
		return "", errors.New("payment secret authentication failed")
	}
	return string(plaintext), nil
}

func IsEncryptedPaymentSecret(value string) bool {
	return strings.HasPrefix(value, paymentSecretPrefix)
}
