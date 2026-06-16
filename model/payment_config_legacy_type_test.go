package model

import "testing"

func TestCreatePaymentConfigWritesLegacyRequiredTypeColumn(t *testing.T) {
	withPaymentConfigTestDB(t)

	config := &PaymentConfig{
		Provider: PaymentProviderWechat,
		Name:     "WeChat Pay",
		Enabled:  true,
	}
	if err := CreatePaymentConfig(config); err != nil {
		t.Fatalf("CreatePaymentConfig returned error: %v", err)
	}

	var legacyType string
	if err := DB.Table("payment_configs").Select("type").Where("provider = ?", PaymentProviderWechat).Scan(&legacyType).Error; err != nil {
		t.Fatalf("query legacy type: %v", err)
	}
	if legacyType != "" {
		t.Fatalf("legacy type = %q, want empty string", legacyType)
	}
}
