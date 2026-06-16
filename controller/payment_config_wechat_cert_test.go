package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const testWechatCertPEM = `-----BEGIN CERTIFICATE-----
MIIBszCCAVmgAwIBAgIUYizvhbqZO9oRP6HjnGXjijLw7PAwDQYJKoZIhvcNAQEL
BQAwEjEQMA4GA1UEAwwHVGVzdCBDQTAeFw0yNjA2MTYwMDAwMDBaFw0yNzA2MTYw
MDAwMDBaMBIxEDAOBgNVBAMMB1Rlc3QgQ0EwXDANBgkqhkiG9w0BAQEFAANLADBI
AkEAu8rlCLLYDlVnmg17D6F0qTKPUbIf933QrvUY5lYsncrW/f1NwZiXXXdKxMOL
068XyhSqVfttCjgpjIb3QIDAQABo1MwUTAdBgNVHQ4EFgQUeJ3qG2uRBAf1wllc
EoYrM3Uf4iQwHwYDVR0jBBgwFoAUeJ3qG2uRBAf1wllcEoYrM3Uf4iQwDwYDVR0T
AQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAANBAFkRS0FPvE7ZtOvwQLeZpAxUbrh4
NQSaSEEXAMPLEONLYmN8P4o8y9YcS0dG8uBqmm0VaK7o4CKq8wrB81YyKec=
-----END CERTIFICATE-----`

func TestCreateWechatPaymentConfigEncryptsAndMasksWechatCert(t *testing.T) {
	setupPaymentConfigControllerTest(t)

	body := model.PaymentConfig{
		Provider:   model.PaymentProviderWechat,
		Name:       "WeChat Pay",
		Enabled:    true,
		WechatCert: testWechatCertPEM,
	}
	ctx, recorder := paymentConfigContext(t, http.MethodPost, "/api/payment-config/", body)

	CreatePaymentConfig(ctx)

	response := decodePaymentConfigResponse(t, recorder)
	if !response.Success {
		t.Fatalf("CreatePaymentConfig response not successful: %+v", response)
	}
	if !common.IsMaskedSecret(response.Data.WechatCert) || response.Data.WechatCert == testWechatCertPEM {
		t.Fatalf("response wechat cert = %q, want masked", response.Data.WechatCert)
	}
	stored, err := model.GetPaymentConfigByProvider(model.PaymentProviderWechat)
	if err != nil {
		t.Fatalf("GetPaymentConfigByProvider: %v", err)
	}
	if stored.WechatCert == testWechatCertPEM {
		t.Fatal("stored wechat cert was not encrypted")
	}
	decrypted, err := common.DecryptPaymentSecret(stored.WechatCert)
	if err != nil {
		t.Fatalf("DecryptPaymentSecret: %v", err)
	}
	if decrypted != testWechatCertPEM {
		t.Fatalf("decrypted wechat cert mismatch")
	}
}
