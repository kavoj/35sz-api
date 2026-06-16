package controller

import "testing"

func TestFilterLegacyEpayPaymentMethodsRemovesEpayMethodsWhenEpayDisabled(t *testing.T) {
	methods := []map[string]string{
		{"name": "支付宝", "type": "alipay"},
		{"name": "微信支付", "type": "wxpay"},
		{"name": "Custom", "type": "custom"},
	}

	filtered := filterLegacyEpayPaymentMethods(methods, false)

	if len(filtered) != 0 {
		t.Fatalf("filtered = %+v, want empty when Epay is disabled", filtered)
	}
}

func TestFilterLegacyEpayPaymentMethodsKeepsMethodsWhenEpayEnabled(t *testing.T) {
	methods := []map[string]string{{"name": "支付宝", "type": "alipay"}}

	filtered := filterLegacyEpayPaymentMethods(methods, true)

	if len(filtered) != 1 || filtered[0]["type"] != "alipay" {
		t.Fatalf("filtered = %+v, want original methods", filtered)
	}
}
