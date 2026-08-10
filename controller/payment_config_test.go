package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func usePaymentConfigControllerTestDB(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	previousCryptoSecret := common.CryptoSecret
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PaymentConfig{}))
	model.DB = db
	common.CryptoSecret = "controller-payment-config-test-secret"
	t.Cleanup(func() {
		model.DB = previousDB
		common.CryptoSecret = previousCryptoSecret
	})
}

func paymentConfigRequest(t *testing.T, method string, path string, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	payload, err := common.Marshal(body)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	return ctx, recorder
}

func decodePaymentConfigResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var response map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestCreatePaymentConfigEncryptsSecretsAndReturnsOnlyConfiguredFlags(t *testing.T) {
	usePaymentConfigControllerTestDB(t)
	ctx, recorder := paymentConfigRequest(t, http.MethodPost, "/api/payment-config", map[string]any{
		"provider":          model.PaymentProviderAlipay,
		"display_name":      "支付宝",
		"app_id":            "app-1",
		"app_private_key":   "merchant-private-key",
		"alipay_public_key": "alipay-public-key",
		"enabled":           true,
	})

	CreatePaymentConfig(ctx)

	response := decodePaymentConfigResponse(t, recorder)
	assert.Equal(t, true, response["success"])
	assert.NotContains(t, recorder.Body.String(), "merchant-private-key")
	data := response["data"].(map[string]any)
	assert.Equal(t, true, data["app_private_key_configured"])
	assert.NotContains(t, data, "app_private_key")

	stored, err := model.GetPaymentConfigByProvider(model.PaymentProviderAlipay)
	require.NoError(t, err)
	assert.True(t, common.IsEncryptedPaymentSecret(stored.AppPrivateKey))
	plaintext, err := common.DecryptPaymentSecret(stored.AppPrivateKey)
	require.NoError(t, err)
	assert.Equal(t, "merchant-private-key", plaintext)
}

func TestUpdatePaymentConfigPreservesOmittedSecretAndClearsOnlyExplicitField(t *testing.T) {
	usePaymentConfigControllerTestDB(t)
	encrypted, err := common.EncryptPaymentSecret("wechat-api-v3-key")
	require.NoError(t, err)
	config := &model.PaymentConfig{
		Provider: model.PaymentProviderWechat, DisplayName: "微信支付",
		WechatAPIKey: encrypted,
	}
	require.NoError(t, model.CreatePaymentConfig(config))

	ctx, recorder := paymentConfigRequest(t, http.MethodPut, "/api/payment-config/1", map[string]any{
		"display_name": "微信支付直连",
	})
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	UpdatePaymentConfig(ctx)
	assert.Equal(t, true, decodePaymentConfigResponse(t, recorder)["success"])
	stored, err := model.GetPaymentConfigByID(config.ID)
	require.NoError(t, err)
	assert.Equal(t, encrypted, stored.WechatAPIKey)

	ctx, recorder = paymentConfigRequest(t, http.MethodPut, "/api/payment-config/1", map[string]any{
		"clear_secrets": []string{"wechat_api_key"},
	})
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	UpdatePaymentConfig(ctx)
	assert.Equal(t, true, decodePaymentConfigResponse(t, recorder)["success"])
	stored, err = model.GetPaymentConfigByID(config.ID)
	require.NoError(t, err)
	assert.Empty(t, stored.WechatAPIKey)
}

func TestUpdatePaymentConfigSecretOnlyPreservesOmittedNonSecrets(t *testing.T) {
	usePaymentConfigControllerTestDB(t)
	config := &model.PaymentConfig{
		Provider: model.PaymentProviderAlipay, DisplayName: "支付宝", Icon: "alipay",
		Enabled: true, SortOrder: 7, AppID: "app-original", GatewayURL: "https://gateway.example",
		NotifyURL: "https://notify.example", ReturnURL: "https://return.example",
		AlipayPublicKey: "existing-public-key",
	}
	require.NoError(t, model.CreatePaymentConfig(config))

	ctx, recorder := paymentConfigRequest(t, http.MethodPut, "/api/payment-config/1", map[string]any{
		"app_private_key": "rotated-private-key",
	})
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	UpdatePaymentConfig(ctx)
	assert.Equal(t, true, decodePaymentConfigResponse(t, recorder)["success"])

	stored, err := model.GetPaymentConfigByID(config.ID)
	require.NoError(t, err)
	assert.Equal(t, "支付宝", stored.DisplayName)
	assert.Equal(t, "alipay", stored.Icon)
	assert.True(t, stored.Enabled)
	assert.Equal(t, 7, stored.SortOrder)
	assert.Equal(t, "app-original", stored.AppID)
	assert.Equal(t, "https://gateway.example", stored.GatewayURL)
	assert.Equal(t, "https://notify.example", stored.NotifyURL)
	assert.Equal(t, "https://return.example", stored.ReturnURL)
	plaintext, err := common.DecryptPaymentSecret(stored.AppPrivateKey)
	require.NoError(t, err)
	assert.Equal(t, "rotated-private-key", plaintext)
}

func TestUpdatePaymentConfigRejectsUnknownSecretClearField(t *testing.T) {
	usePaymentConfigControllerTestDB(t)
	config := &model.PaymentConfig{Provider: model.PaymentProviderAlipay, DisplayName: "支付宝"}
	require.NoError(t, model.CreatePaymentConfig(config))

	ctx, recorder := paymentConfigRequest(t, http.MethodPut, "/api/payment-config/1", map[string]any{
		"clear_secrets": []string{"not_a_secret"},
	})
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	UpdatePaymentConfig(ctx)

	response := decodePaymentConfigResponse(t, recorder)
	assert.Equal(t, false, response["success"])
	stored, err := model.GetPaymentConfigByID(config.ID)
	require.NoError(t, err)
	assert.Equal(t, "支付宝", stored.DisplayName)
}

func TestUpdatePaymentConfigRejectsProviderChange(t *testing.T) {
	usePaymentConfigControllerTestDB(t)
	config := &model.PaymentConfig{Provider: model.PaymentProviderAlipay, DisplayName: "支付宝"}
	require.NoError(t, model.CreatePaymentConfig(config))

	ctx, recorder := paymentConfigRequest(t, http.MethodPut, "/api/payment-config/1", map[string]any{
		"provider": model.PaymentProviderWechat,
	})
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	UpdatePaymentConfig(ctx)

	response := decodePaymentConfigResponse(t, recorder)
	assert.Equal(t, false, response["success"])
	stored, err := model.GetPaymentConfigByID(config.ID)
	require.NoError(t, err)
	assert.Equal(t, model.PaymentProviderAlipay, stored.Provider)
}

func TestPaymentConfigCannotBeEnabledUntilProviderFieldsAreComplete(t *testing.T) {
	usePaymentConfigControllerTestDB(t)

	ctx, recorder := paymentConfigRequest(t, http.MethodPost, "/api/payment-config", map[string]any{
		"provider": model.PaymentProviderAlipay,
		"enabled":  true,
		"app_id":   "app-1",
	})
	CreatePaymentConfig(ctx)
	assert.Equal(t, false, decodePaymentConfigResponse(t, recorder)["success"])

	config := &model.PaymentConfig{Provider: model.PaymentProviderWechat, DisplayName: "微信支付"}
	require.NoError(t, model.CreatePaymentConfig(config))
	ctx, recorder = paymentConfigRequest(t, http.MethodPut, "/api/payment-config/1", map[string]any{
		"enabled": true,
	})
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	UpdatePaymentConfig(ctx)
	assert.Equal(t, false, decodePaymentConfigResponse(t, recorder)["success"])
	stored, err := model.GetPaymentConfigByID(config.ID)
	require.NoError(t, err)
	assert.False(t, stored.Enabled)
}
