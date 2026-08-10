package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDecryptedPaymentConfigReturnsPlaintextOnlyForEnabledProvider(t *testing.T) {
	usePaymentConfigControllerTestDB(t)
	privateKey, err := common.EncryptPaymentSecret("merchant-private-key")
	require.NoError(t, err)
	publicKey, err := common.EncryptPaymentSecret("alipay-public-key")
	require.NoError(t, err)
	config := &model.PaymentConfig{
		Provider: model.PaymentProviderAlipay, Enabled: true, AppID: "app-1",
		AppPrivateKey: privateKey, AlipayPublicKey: publicKey,
	}
	require.NoError(t, model.CreatePaymentConfig(config))

	decrypted, err := getDecryptedPaymentConfig(model.PaymentProviderAlipay)
	require.NoError(t, err)
	assert.Equal(t, "merchant-private-key", decrypted.AppPrivateKey)
	assert.Equal(t, "alipay-public-key", decrypted.AlipayPublicKey)

	require.NoError(t, model.UpdatePaymentConfigFields(config.ID, map[string]any{"enabled": false}))
	_, err = getDecryptedPaymentConfig(model.PaymentProviderAlipay)
	require.Error(t, err)
}
