package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func TestGetPayMoneyInCNYDisplayUsesEnteredAmountAsPaymentAmount(t *testing.T) {
	oldPrice := operation_setting.Price
	oldDisplay := operation_setting.GetQuotaDisplayType()
	t.Cleanup(func() {
		operation_setting.Price = oldPrice
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplay
	})

	operation_setting.Price = 7.3
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeCNY

	payMoney := getPayMoney(200, "")
	if payMoney != 200 {
		t.Fatalf("getPayMoney(200) = %v, want 200", payMoney)
	}
}

func TestGetPayMoneyInUSDDisplayMultipliesByPrice(t *testing.T) {
	oldPrice := operation_setting.Price
	oldDisplay := operation_setting.GetQuotaDisplayType()
	t.Cleanup(func() {
		operation_setting.Price = oldPrice
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplay
	})

	operation_setting.Price = 7.3
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD

	payMoney := getPayMoney(10, "")
	if payMoney != 73 {
		t.Fatalf("getPayMoney(10) = %v, want 73", payMoney)
	}
}
