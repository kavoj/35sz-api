package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
)

type SubscriptionAlipayPayRequest struct {
	PlanID        int    `json:"plan_id"`
	PaymentMethod string `json:"payment_method"`
}

func SubscriptionRequestAlipayPay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	var req SubscriptionAlipayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanID <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.PaymentMethod == "" {
		req.PaymentMethod = model.PaymentMethodAlipayPC
	}
	if req.PaymentMethod != model.PaymentMethodAlipayPC && req.PaymentMethod != model.PaymentMethodAlipayH5 {
		common.ApiErrorMsg(c, "不支持的支付宝支付方式")
		return
	}
	plan, err := model.GetSubscriptionPlanById(req.PlanID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled || plan.PriceAmount < 0.01 {
		common.ApiErrorMsg(c, "套餐不可购买")
		return
	}
	userID := c.GetInt("id")
	if plan.MaxPurchasePerUser > 0 {
		count, err := model.CountUserSubscriptionsByPlan(userID, plan.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			common.ApiErrorMsg(c, "已达到该套餐购买上限")
			return
		}
	}
	config, err := getDecryptedPaymentConfig(model.PaymentProviderAlipay)
	if err != nil {
		common.ApiErrorMsg(c, "支付宝支付未配置")
		return
	}
	tradeNo := fmt.Sprintf("SUB-ALIPAY-%d-%d-%s", userID, time.Now().UnixMilli(), randstr.String(6))
	order := &model.SubscriptionOrder{UserId: userID, PlanId: plan.Id, Money: plan.PriceAmount, TradeNo: tradeNo, PaymentMethod: req.PaymentMethod, PaymentProvider: model.PaymentProviderAlipay, CreateTime: time.Now().Unix(), Status: common.TopUpStatusPending}
	if err := order.Insert(); err != nil {
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}
	returnURL := paymentReturnPath("/wallet?show_history=true")
	client, err := service.NewAlipayPayClient(config, "/api/subscription/alipay/notify", returnURL)
	if err != nil {
		_ = model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderAlipay)
		common.ApiErrorMsg(c, "支付配置错误")
		return
	}
	expireTime := time.Now().Add(30 * time.Minute).Format("2006-01-02 15:04:05")
	var payURL string
	if req.PaymentMethod == model.PaymentMethodAlipayPC {
		payURL, err = client.CreatePagePay(tradeNo, fmt.Sprintf("订阅套餐:%s", plan.Title), strconv.FormatFloat(plan.PriceAmount, 'f', 2, 64), expireTime)
	} else {
		payURL, err = client.CreateWAPPay(tradeNo, fmt.Sprintf("订阅套餐:%s", plan.Title), strconv.FormatFloat(plan.PriceAmount, 'f', 2, 64), expireTime, returnURL)
	}
	if err != nil {
		_ = model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderAlipay)
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"pay_url": payURL, "trade_no": tradeNo}})
}
