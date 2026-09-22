package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/thanhpk/randstr"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
)

type WechatPayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

func WechatNotify(c *gin.Context) {
	config, err := getDecryptedPaymentConfig(model.PaymentProviderWechat)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "config unavailable"})
		return
	}
	client, err := service.NewWechatPayClient(config, "/api/wechat/notify")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "config invalid"})
		return
	}
	transaction := new(payments.Transaction)
	if _, err := client.ParseNotifyRequest(c.Request.Context(), c.Request, transaction); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "verify failed"})
		return
	}
	if transaction.OutTradeNo == nil || transaction.TradeState == nil {
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "invalid notification"})
		return
	}
	tradeNo := *transaction.OutTradeNo
	if *transaction.TradeState == "CLOSED" || *transaction.TradeState == "PAYERROR" {
		_ = model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderWechat, common.TopUpStatusFailed)
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "成功"})
		return
	}
	if *transaction.TradeState != "SUCCESS" {
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "成功"})
		return
	}
	if transaction.Amount == nil || transaction.Amount.Total == nil || *transaction.Amount.Total <= 0 {
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "invalid amount"})
		return
	}
	if err := model.RechargeWechatPaid(tradeNo, *transaction.Amount.Total, c.ClientIP()); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "recharge failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "成功"})
}

func RequestWechatPay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	var req WechatPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.PaymentMethod != model.PaymentMethodWechatNative && req.PaymentMethod != model.PaymentMethodWechatH5 {
		common.ApiErrorMsg(c, "不支持的微信支付方式")
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
	config, err := getDecryptedPaymentConfig(model.PaymentProviderWechat)
	if err != nil {
		common.ApiErrorMsg(c, "微信支付未配置")
		return
	}
	tradeNo := fmt.Sprintf("WECHAT-%d-%d-%s", userID, time.Now().UnixMilli(), randstr.String(6))
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = decimal.NewFromInt(amount).Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	}
	order := &model.TopUp{UserId: userID, Amount: amount, Money: payMoney, TradeNo: tradeNo, PaymentMethod: req.PaymentMethod, PaymentProvider: model.PaymentProviderWechat, CreateTime: time.Now().Unix(), Status: common.TopUpStatusPending}
	if err := order.Insert(); err != nil {
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}
	client, err := service.NewWechatPayClient(config, "/api/wechat/notify")
	if err != nil {
		_ = model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderWechat, common.TopUpStatusFailed)
		common.ApiErrorMsg(c, "支付配置错误")
		return
	}
	expires := time.Now().Add(30 * time.Minute)
	var payURL string
	if req.PaymentMethod == model.PaymentMethodWechatNative {
		payURL, err = client.CreateNativeOrder(c.Request.Context(), tradeNo, fmt.Sprintf("账户充值%d", req.Amount), service.YuanToFen(payMoney), expires)
	} else {
		payURL, err = client.CreateH5Order(c.Request.Context(), tradeNo, fmt.Sprintf("账户充值%d", req.Amount), service.YuanToFen(payMoney), expires, c.ClientIP())
	}
	if err != nil {
		_ = model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderWechat, common.TopUpStatusFailed)
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"pay_url": payURL, "trade_no": tradeNo}})
}
