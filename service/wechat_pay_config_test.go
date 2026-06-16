package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
)

func TestValidateWechatPayConfigPublicKeyModeDoesNotRequireAPIv3Key(t *testing.T) {
	config := &model.PaymentConfig{
		WechatAuthMode:    "public_key",
		WechatAppID:       "wx-app",
		WechatMchID:       "1746971547",
		WechatSerialNo:    "662CEF85BA993BDA113FA1E39C65E38A32F0ECF0",
		WechatPrivateKey:  "private-key",
		WechatPublicKeyID: "PUB_KEY_ID_0117469715472026061500382161001801",
		WechatPublicKey:   testWechatPayPublicKey,
	}

	if err := validateWechatPayConfig(config); err != nil {
		t.Fatalf("validateWechatPayConfig returned error: %v", err)
	}
}

func TestValidateWechatPayConfigCertificateModeRequiresAPIv3Key(t *testing.T) {
	config := &model.PaymentConfig{
		WechatAuthMode:   "certificate",
		WechatAppID:      "wx-app",
		WechatMchID:      "1746971547",
		WechatSerialNo:   "662CEF85BA993BDA113FA1E39C65E38A32F0ECF0",
		WechatPrivateKey: "private-key",
	}

	err := validateWechatPayConfig(config)
	if err == nil || !strings.Contains(err.Error(), "api v3 key") {
		t.Fatalf("validateWechatPayConfig error = %v, want api v3 key error", err)
	}
}
