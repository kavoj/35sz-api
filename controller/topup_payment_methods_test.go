package controller

import "testing"

func TestFilterLegacyNativePaymentMethodsRemovesLegacyWechatWhenNativeWechatEnabled(t *testing.T) {
	methods := []map[string]string{
		{"name": "微信支付", "type": "wxpay"},
		{"name": "支付宝", "type": "alipay"},
		{"name": "Custom", "type": "custom"},
	}

	filtered := filterLegacyNativePaymentMethods(methods, true, false)

	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2: %+v", len(filtered), filtered)
	}
	for _, method := range filtered {
		if method["type"] == "wxpay" {
			t.Fatalf("legacy wechat method was not removed: %+v", filtered)
		}
	}
}

func TestFilterLegacyNativePaymentMethodsRemovesLegacyAlipayWhenNativeAlipayEnabled(t *testing.T) {
	methods := []map[string]string{
		{"name": "微信支付", "type": "wxpay"},
		{"name": "支付宝", "type": "alipay"},
		{"name": "Custom", "type": "custom"},
	}

	filtered := filterLegacyNativePaymentMethods(methods, false, true)

	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2: %+v", len(filtered), filtered)
	}
	for _, method := range filtered {
		if method["type"] == "alipay" {
			t.Fatalf("legacy alipay method was not removed: %+v", filtered)
		}
	}
}
