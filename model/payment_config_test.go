package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func usePaymentConfigTestDB(t *testing.T) {
	t.Helper()
	previous := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&PaymentConfig{}))
	DB = db
	t.Cleanup(func() { DB = previous })
}

func TestPaymentConfigProviderIsUniqueAndSecretsAreNotJSONSerializable(t *testing.T) {
	usePaymentConfigTestDB(t)

	config := &PaymentConfig{
		Provider:      PaymentProviderAlipay,
		DisplayName:   "支付宝",
		AppPrivateKey: "encrypted-private-key",
	}
	require.NoError(t, CreatePaymentConfig(config))
	require.Error(t, CreatePaymentConfig(&PaymentConfig{Provider: PaymentProviderAlipay}))

	view := config.PublicView()
	assert.True(t, view.AppPrivateKeyConfigured)
	assert.Empty(t, view.AppPrivateKey)
}

func TestUpdatePaymentConfigFieldsPreservesOmittedSecret(t *testing.T) {
	usePaymentConfigTestDB(t)

	config := &PaymentConfig{Provider: PaymentProviderWechat, WechatAPIKey: "ciphertext"}
	require.NoError(t, CreatePaymentConfig(config))
	require.NoError(t, UpdatePaymentConfigFields(config.ID, map[string]any{"display_name": "微信支付"}))

	stored, err := GetPaymentConfigByID(config.ID)
	require.NoError(t, err)
	assert.Equal(t, "ciphertext", stored.WechatAPIKey)
	assert.Equal(t, "微信支付", stored.DisplayName)
}
