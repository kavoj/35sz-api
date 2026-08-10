package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRechargeAlipayRequiresMatchingPendingOrderAndIsIdempotent(t *testing.T) {
	truncateTables(t)
	insertUserForPaymentGuardTest(t, 601, 0)
	insertTopUpForPaymentGuardTest(t, "alipay-direct-order", 601, PaymentProviderAlipay)

	require.NoError(t, RechargeAlipay("alipay-direct-order", "127.0.0.1"))
	assert.Equal(t, common.TopUpStatusSuccess, getTopUpStatusForPaymentGuardTest(t, "alipay-direct-order"))
	assert.Greater(t, getUserQuotaForPaymentGuardTest(t, 601), 0)
	quota := getUserQuotaForPaymentGuardTest(t, 601)

	require.NoError(t, RechargeAlipay("alipay-direct-order", "127.0.0.1"))
	assert.Equal(t, quota, getUserQuotaForPaymentGuardTest(t, 601))
}

func TestRechargeAlipayRejectsWechatOrder(t *testing.T) {
	truncateTables(t)
	insertUserForPaymentGuardTest(t, 602, 0)
	insertTopUpForPaymentGuardTest(t, "alipay-cross-gateway", 602, PaymentProviderWechat)

	err := RechargeAlipay("alipay-cross-gateway", "127.0.0.1")
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, "alipay-cross-gateway"))
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 602))
}

func TestRechargeWechatRequiresMatchingPendingOrderAndIsIdempotent(t *testing.T) {
	truncateTables(t)
	insertUserForPaymentGuardTest(t, 603, 0)
	insertTopUpForPaymentGuardTest(t, "wechat-direct-order", 603, PaymentProviderWechat)

	require.NoError(t, RechargeWechat("wechat-direct-order", "127.0.0.1"))
	quota := getUserQuotaForPaymentGuardTest(t, 603)
	assert.Greater(t, quota, 0)
	require.NoError(t, RechargeWechat("wechat-direct-order", "127.0.0.1"))
	assert.Equal(t, quota, getUserQuotaForPaymentGuardTest(t, 603))
}

func TestRechargeWechatRejectsAlipayOrder(t *testing.T) {
	truncateTables(t)
	insertUserForPaymentGuardTest(t, 604, 0)
	insertTopUpForPaymentGuardTest(t, "wechat-cross-gateway", 604, PaymentProviderAlipay)

	err := RechargeWechat("wechat-cross-gateway", "127.0.0.1")
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, "wechat-cross-gateway"))
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 604))
}
