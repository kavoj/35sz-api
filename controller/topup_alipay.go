package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/thanhpk/randstr"
)

type AlipayPayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

func isAlipayTopUpEnabled() bool {
	if !isPaymentComplianceConfirmed() {
		return false
	}
	_, err := model.GetEnabledPaymentConfigByProvider(model.PaymentProviderAlipay)
	return err == nil
}

func RequestAlipayPay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	var req AlipayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.PaymentMethod != model.PaymentMethodAlipayPC && req.PaymentMethod != model.PaymentMethodAlipayH5 {
		common.ApiErrorMsg(c, "不支持的支付宝支付方式")
		return
	}
	if req.Amount < getMinTopup() {
		common.ApiErrorMsg(c, fmt.Sprintf("充值数量不能小于 %d", getMinTopup()))
		return
	}
	userID := c.GetInt("id")
	group, err := model.GetUserGroup(userID, true)
	if err != nil {
		common.ApiErrorMsg(c, "获取用户分组失败")
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		common.ApiErrorMsg(c, "充值金额过低")
		return
	}
	config, err := getDecryptedPaymentConfig(model.PaymentProviderAlipay)
	if err != nil {
		common.ApiErrorMsg(c, "支付宝支付未配置")
		return
	}
	tradeNo := fmt.Sprintf("ALIPAY-%d-%d-%s", userID, time.Now().UnixMilli(), randstr.String(6))
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = decimal.NewFromInt(amount).Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	}
	order := &model.TopUp{UserId: userID, Amount: amount, Money: payMoney, TradeNo: tradeNo, PaymentMethod: req.PaymentMethod, PaymentProvider: model.PaymentProviderAlipay, CreateTime: time.Now().Unix(), Status: common.TopUpStatusPending}
	if err := order.Insert(); err != nil {
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}
	returnURL := paymentReturnPath("/wallet?show_history=true")
	client, err := service.NewAlipayPayClient(config, "/api/alipay/notify", returnURL)
	if err != nil {
		_ = model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderAlipay, common.TopUpStatusFailed)
		common.ApiErrorMsg(c, "支付配置错误")
		return
	}
	totalAmount := strconv.FormatFloat(payMoney, 'f', 2, 64)
	expireTime := time.Now().Add(30 * time.Minute).Format("2006-01-02 15:04:05")
	var payURL string
	if req.PaymentMethod == model.PaymentMethodAlipayPC {
		payURL, err = client.CreatePagePay(tradeNo, fmt.Sprintf("账户充值%d", req.Amount), totalAmount, expireTime)
	} else {
		payURL, err = client.CreateWAPPay(tradeNo, fmt.Sprintf("账户充值%d", req.Amount), totalAmount, expireTime, returnURL)
	}
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝拉起失败 trade_no=%s error=%q", tradeNo, err.Error()))
		_ = model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderAlipay, common.TopUpStatusFailed)
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"pay_url": payURL, "trade_no": tradeNo}})
}

func RequestAlipayAmount(c *gin.Context) {
	var req AlipayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount < getMinTopup() {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	group, err := model.GetUserGroup(c.GetInt("id"), true)
	if err != nil {
		common.ApiErrorMsg(c, "获取用户分组失败")
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		common.ApiErrorMsg(c, "充值金额过低")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": strconv.FormatFloat(payMoney, 'f', 2, 64)})
}
