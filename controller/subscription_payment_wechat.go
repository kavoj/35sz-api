package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
)

type SubscriptionWechatPayRequest struct {
	PlanID        int    `json:"plan_id"`
	PaymentMethod string `json:"payment_method"`
}

func SubscriptionRequestWechatPay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	var req SubscriptionWechatPayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanID <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.PaymentMethod == "" {
		req.PaymentMethod = model.PaymentMethodWechatNative
	}
	if req.PaymentMethod != model.PaymentMethodWechatNative && req.PaymentMethod != model.PaymentMethodWechatH5 {
		common.ApiErrorMsg(c, "不支持的微信支付方式")
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
	config, err := getDecryptedPaymentConfig(model.PaymentProviderWechat)
	if err != nil {
		common.ApiErrorMsg(c, "微信支付未配置")
		return
	}
	tradeNo := fmt.Sprintf("SUB-WECHAT-%d-%d-%s", userID, time.Now().UnixMilli(), randstr.String(6))
	order := &model.SubscriptionOrder{UserId: userID, PlanId: plan.Id, Money: plan.PriceAmount, TradeNo: tradeNo, PaymentMethod: req.PaymentMethod, PaymentProvider: model.PaymentProviderWechat, CreateTime: time.Now().Unix(), Status: common.TopUpStatusPending}
	if err := order.Insert(); err != nil {
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}
	client, err := service.NewWechatPayClient(config, "/api/subscription/wechat/notify")
	if err != nil {
		_ = model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderWechat)
		common.ApiErrorMsg(c, "支付配置错误")
		return
	}
	expires := time.Now().Add(30 * time.Minute)
	var payURL string
	if req.PaymentMethod == model.PaymentMethodWechatNative {
		payURL, err = client.CreateNativeOrder(c.Request.Context(), tradeNo, "订阅套餐:"+plan.Title, service.YuanToFen(plan.PriceAmount), expires)
	} else {
		payURL, err = client.CreateH5Order(c.Request.Context(), tradeNo, "订阅套餐:"+plan.Title, service.YuanToFen(plan.PriceAmount), expires, c.ClientIP())
	}
	if err != nil {
		_ = model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderWechat)
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"pay_url": payURL, "trade_no": tradeNo}})
}
