package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func TestCreateWechatPaymentConfigEncryptsAndMasksAppSecret(t *testing.T) {
	setupPaymentConfigControllerTest(t)

	body := model.PaymentConfig{
		Provider:        model.PaymentProviderWechat,
		Name:            "WeChat Pay",
		Enabled:         true,
		WechatAppID:     "wx-app",
		WechatAppSecret: "wechat-app-secret",
	}
	ctx, recorder := paymentConfigContext(t, http.MethodPost, "/api/payment-config/", body)

	CreatePaymentConfig(ctx)

	response := decodePaymentConfigResponse(t, recorder)
	if !response.Success {
		t.Fatalf("CreatePaymentConfig response not successful: %+v", response)
	}
	if !common.IsMaskedSecret(response.Data.WechatAppSecret) || response.Data.WechatAppSecret == "wechat-app-secret" {
		t.Fatalf("response wechat app secret = %q, want masked", response.Data.WechatAppSecret)
	}
	stored, err := model.GetPaymentConfigByProvider(model.PaymentProviderWechat)
	if err != nil {
		t.Fatalf("GetPaymentConfigByProvider: %v", err)
	}
	if stored.WechatAppSecret == "wechat-app-secret" {
		t.Fatal("stored wechat app secret was not encrypted")
	}
	decrypted, err := common.DecryptPaymentSecret(stored.WechatAppSecret)
	if err != nil {
		t.Fatalf("DecryptPaymentSecret: %v", err)
	}
	if decrypted != "wechat-app-secret" {
		t.Fatalf("decrypted wechat app secret = %q", decrypted)
	}
}
