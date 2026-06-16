package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func TestRechargeWechatInCNYDisplayAddsQuotaFromPaymentAmountDividedByPrice(t *testing.T) {
	truncateTables(t)
	oldPrice := operation_setting.Price
	oldDisplay := operation_setting.GetQuotaDisplayType()
	t.Cleanup(func() {
		operation_setting.Price = oldPrice
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplay
	})
	operation_setting.Price = 7.3
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeCNY

	insertUserForPaymentGuardTest(t, 301, 0)
	topUp := &TopUp{
		UserId:          301,
		Amount:          73,
		Money:           73,
		TradeNo:         "wechat-cny-73",
		PaymentMethod:   PaymentMethodWechatNative,
		PaymentProvider: PaymentProviderWechat,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())

	require.NoError(t, RechargeWechat("wechat-cny-73", "127.0.0.1"))

	quota := getUserQuotaForPaymentGuardTest(t, 301)
	expected := int(10 * common.QuotaPerUnit)
	require.Equal(t, expected, quota)
}
