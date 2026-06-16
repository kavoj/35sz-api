package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

func TestParseWechatMerchantCertSerialReturnsUppercaseHex(t *testing.T) {
	serialNumber := new(big.Int)
	serialNumber.SetString("662CEF85BA993BDA113FA1E39C65E38A32F0ECF0", 16)
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{
			SerialNumber: serialNumber,
			NotBefore:    time.Now(),
			NotAfter:     time.Now().Add(time.Hour),
		},
		&x509.Certificate{SerialNumber: serialNumber},
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))

	serial, err := parseWechatMerchantCertSerial(certPEM)
	if err != nil {
		t.Fatalf("parseWechatMerchantCertSerial returned error: %v", err)
	}
	if serial != "662CEF85BA993BDA113FA1E39C65E38A32F0ECF0" {
		t.Fatalf("serial = %q", serial)
	}
}
