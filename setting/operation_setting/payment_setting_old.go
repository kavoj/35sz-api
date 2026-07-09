/**
此文件为旧版支付设置文件，如需增加新的参数、变量等，请在 payment_setting.go 中添加
This file is the old version of the payment settings file. If you need to add new parameters, variables, etc., please add them in payment_setting.go
*/

package operation_setting

import (
	"github.com/QuantumNous/new-api/common"
)

var PayAddress = ""
var CustomCallbackAddress = ""
var EpayId = ""
var EpayKey = ""
var Price = 7.3
var MinTopUp = 1
var USDExchangeRate = 7.3

// RechargePremium is the platform's surcharge multiplier applied on top of
// USDExchangeRate at recharge time. Introduced by PR-4 (CNY reconciliation)
// to make the historically-conflated "Price" concept explicit:
//
//	Price effective for recharge  =  USDExchangeRate * RechargePremium
//
// Default 1.0 means "no premium, charge at the raw exchange rate". Setting
// it to 1.05 collects a 5% platform fee on top of the market rate. This
// value is snapshotted on the TopUp row at payment time so historical
// reconciliation is unaffected by later admin changes.
//
// Historically, `Price` and `USDExchangeRate` were two independent settings
// both defaulting to 7.3, and drift between them silently leaked CNY (see
// db-currency-storage-audit report). RechargePremium replaces the implicit
// divergence with a first-class field the admin can see and reason about.
var RechargePremium = 1.0


var PayMethods = []map[string]string{
	{
		"name": "支付宝",
		"icon": "SiAlipay",
		"type": "alipay",
	},
	{
		"name": "微信",
		"icon": "SiWechat",
		"type": "wxpay",
	},
	{
		"name":      "自定义1",
		"icon":      "LuCreditCard",
		"type":      "custom1",
		"min_topup": "50",
	},
}

func UpdatePayMethodsByJsonString(jsonString string) error {
	PayMethods = make([]map[string]string, 0)
	return common.Unmarshal([]byte(jsonString), &PayMethods)
}

func PayMethods2JsonString() string {
	jsonBytes, err := common.Marshal(PayMethods)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

func ContainsPayMethod(method string) bool {
	for _, payMethod := range PayMethods {
		if payMethod["type"] == method {
			return true
		}
	}
	return false
}
