package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWechatPayConfigEnforcesAuthModeRequirements(t *testing.T) {
	base := &model.PaymentConfig{WechatAppID: "wx-app", WechatMchID: "mch", WechatSerialNo: "serial", WechatPrivateKey: "private"}
	require.Error(t, validateWechatPayConfig(base))

	certificate := *base
	certificate.WechatAPIKey = "api-v3"
	require.NoError(t, validateWechatPayConfig(&certificate))

	publicKey := *base
	publicKey.WechatAuthMode = "public_key"
	publicKey.WechatPublicKeyID = "PUB_KEY_ID"
	publicKey.WechatPublicKey = "public-key"
	require.NoError(t, validateWechatPayConfig(&publicKey))
}

func TestYuanToFenRoundsToNearestFen(t *testing.T) {
	assert.Equal(t, int64(1), YuanToFen(0.01))
	assert.Equal(t, int64(1001), YuanToFen(10.005))
}
