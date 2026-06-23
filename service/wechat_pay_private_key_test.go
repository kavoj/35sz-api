package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
)

func generatePKCS8KeyPEM(t *testing.T) string {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func TestLoadWechatPrivateKeyAcceptsValidPEM(t *testing.T) {
	keyPEM := generatePKCS8KeyPEM(t)
	if _, err := loadWechatPrivateKey(keyPEM); err != nil {
		t.Fatalf("loadWechatPrivateKey(valid) returned error: %v", err)
	}
}

func TestLoadWechatPrivateKeyRecoversLiteralEscapedNewlines(t *testing.T) {
	keyPEM := generatePKCS8KeyPEM(t)
	// Simulate a key that was pasted as a single line with literal "\n"
	// (e.g. copied out of a single-line JSON config).
	broken := strings.ReplaceAll(keyPEM, "\n", "\\n")
	if !strings.Contains(broken, "\\n") {
		t.Fatalf("test setup failed: expected literal backslash-n in broken key")
	}
	if _, err := loadWechatPrivateKey(broken); err != nil {
		t.Fatalf("loadWechatPrivateKey(literal-\\n) returned error: %v", err)
	}
}

func TestLoadWechatPrivateKeyRecoversCRLF(t *testing.T) {
	keyPEM := generatePKCS8KeyPEM(t)
	crlf := strings.ReplaceAll(keyPEM, "\n", "\r\n")
	if _, err := loadWechatPrivateKey(crlf); err != nil {
		t.Fatalf("loadWechatPrivateKey(CRLF) returned error: %v", err)
	}
}

func TestLoadWechatPrivateKeyRejectsGarbage(t *testing.T) {
	if _, err := loadWechatPrivateKey("not a pem at all"); err == nil {
		t.Fatalf("loadWechatPrivateKey(garbage) expected error, got nil")
	}
}
