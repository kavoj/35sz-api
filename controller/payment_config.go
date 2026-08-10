package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type paymentConfigWriteRequest struct {
	Provider    *string `json:"provider"`
	DisplayName *string `json:"display_name"`
	Icon        *string `json:"icon"`
	Enabled     *bool   `json:"enabled"`
	SortOrder   *int    `json:"sort_order"`
	AppID       *string `json:"app_id"`

	AppPrivateKey       *string `json:"app_private_key"`
	AlipayPublicKey     *string `json:"alipay_public_key"`
	AlipayAppPublicCert *string `json:"alipay_app_public_cert"`
	AlipayPublicCert    *string `json:"alipay_public_cert"`
	AlipayRootCert      *string `json:"alipay_root_cert"`

	WechatAppID       *string `json:"wechat_app_id"`
	WechatMchID       *string `json:"wechat_mch_id"`
	WechatSerialNo    *string `json:"wechat_serial_no"`
	WechatAuthMode    *string `json:"wechat_auth_mode"`
	WechatPublicKeyID *string `json:"wechat_public_key_id"`
	WechatAppSecret   *string `json:"wechat_app_secret"`
	WechatAPIKey      *string `json:"wechat_api_key"`
	WechatPrivateKey  *string `json:"wechat_private_key"`
	WechatCert        *string `json:"wechat_cert"`
	WechatPublicKey   *string `json:"wechat_public_key"`

	GatewayURL *string `json:"gateway_url"`
	NotifyURL  *string `json:"notify_url"`
	ReturnURL  *string `json:"return_url"`

	ClearSecrets []string `json:"clear_secrets"`
}

var paymentConfigSecretFields = map[string]struct{}{
	"app_private_key": {}, "alipay_public_key": {}, "alipay_app_public_cert": {},
	"alipay_public_cert": {}, "alipay_root_cert": {}, "wechat_app_secret": {},
	"wechat_api_key": {}, "wechat_private_key": {}, "wechat_cert": {},
	"wechat_public_key": {},
}

func validatePaymentConfigForEnable(config *model.PaymentConfig) error {
	if config == nil || !config.Enabled {
		return nil
	}
	switch config.Provider {
	case model.PaymentProviderAlipay:
		if config.AppID == "" || config.AppPrivateKey == "" {
			return errors.New("alipay requires app_id and app_private_key before enabling")
		}
		if config.AlipayPublicKey == "" && (config.AlipayAppPublicCert == "" || config.AlipayPublicCert == "" || config.AlipayRootCert == "") {
			return errors.New("alipay requires alipay_public_key or complete certificates before enabling")
		}
	case model.PaymentProviderWechat:
		if config.WechatAppID == "" || config.WechatMchID == "" || config.WechatPrivateKey == "" {
			return errors.New("wechat requires wechat_app_id, wechat_mch_id and wechat_private_key before enabling")
		}
		if config.WechatSerialNo == "" && config.WechatCert == "" {
			return errors.New("wechat requires wechat_serial_no or wechat_cert before enabling")
		}
		if config.WechatAuthMode == "public_key" {
			if config.WechatPublicKeyID == "" || config.WechatPublicKey == "" {
				return errors.New("wechat public_key mode requires wechat_public_key_id and wechat_public_key before enabling")
			}
		} else if config.WechatAPIKey == "" {
			return errors.New("wechat requires wechat_api_key before enabling")
		}
	default:
		return errors.New("unsupported payment provider")
	}
	return nil
}

func encryptPaymentSecretValue(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	return common.EncryptPaymentSecret(value)
}

func writePaymentConfigSecrets(req paymentConfigWriteRequest, config *model.PaymentConfig) error {
	secretValues := map[string]*string{
		"app_private_key": req.AppPrivateKey, "alipay_public_key": req.AlipayPublicKey,
		"alipay_app_public_cert": req.AlipayAppPublicCert, "alipay_public_cert": req.AlipayPublicCert,
		"alipay_root_cert": req.AlipayRootCert, "wechat_app_secret": req.WechatAppSecret,
		"wechat_api_key": req.WechatAPIKey, "wechat_private_key": req.WechatPrivateKey,
		"wechat_cert": req.WechatCert, "wechat_public_key": req.WechatPublicKey,
	}
	for field, value := range secretValues {
		if value == nil || strings.TrimSpace(*value) == "" {
			continue
		}
		if strings.HasSuffix(*value, "****") {
			return fmt.Errorf("masked value is not accepted for %s", field)
		}
		encrypted, err := encryptPaymentSecretValue(*value)
		if err != nil {
			return err
		}
		switch field {
		case "app_private_key":
			config.AppPrivateKey = encrypted
		case "alipay_public_key":
			config.AlipayPublicKey = encrypted
		case "alipay_app_public_cert":
			config.AlipayAppPublicCert = encrypted
		case "alipay_public_cert":
			config.AlipayPublicCert = encrypted
		case "alipay_root_cert":
			config.AlipayRootCert = encrypted
		case "wechat_app_secret":
			config.WechatAppSecret = encrypted
		case "wechat_api_key":
			config.WechatAPIKey = encrypted
		case "wechat_private_key":
			config.WechatPrivateKey = encrypted
		case "wechat_cert":
			config.WechatCert = encrypted
		case "wechat_public_key":
			config.WechatPublicKey = encrypted
		}
	}
	return nil
}

func paymentConfigFromRequest(req paymentConfigWriteRequest) (model.PaymentConfig, error) {
	config := model.PaymentConfig{}
	if req.Provider != nil {
		config.Provider = *req.Provider
	}
	if req.DisplayName != nil {
		config.DisplayName = *req.DisplayName
	}
	if req.Icon != nil {
		config.Icon = *req.Icon
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.SortOrder != nil {
		config.SortOrder = *req.SortOrder
	}
	if req.AppID != nil {
		config.AppID = *req.AppID
	}
	if req.WechatAppID != nil {
		config.WechatAppID = *req.WechatAppID
	}
	if req.WechatMchID != nil {
		config.WechatMchID = *req.WechatMchID
	}
	if req.WechatSerialNo != nil {
		config.WechatSerialNo = *req.WechatSerialNo
	}
	if req.WechatAuthMode != nil {
		config.WechatAuthMode = *req.WechatAuthMode
	}
	if req.WechatPublicKeyID != nil {
		config.WechatPublicKeyID = *req.WechatPublicKeyID
	}
	if req.GatewayURL != nil {
		config.GatewayURL = *req.GatewayURL
	}
	if req.NotifyURL != nil {
		config.NotifyURL = *req.NotifyURL
	}
	if req.ReturnURL != nil {
		config.ReturnURL = *req.ReturnURL
	}
	if !model.IsSupportedPaymentConfigProvider(config.Provider) {
		return model.PaymentConfig{}, errors.New("unsupported payment provider")
	}
	if err := writePaymentConfigSecrets(req, &config); err != nil {
		return model.PaymentConfig{}, err
	}
	if err := validatePaymentConfigForEnable(&config); err != nil {
		return model.PaymentConfig{}, err
	}
	return config, nil
}

func CreatePaymentConfig(c *gin.Context) {
	var req paymentConfigWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid payment config")
		return
	}
	if req.Provider != nil {
		if _, err := model.GetPaymentConfigByProvider(*req.Provider); err == nil {
			common.ApiErrorMsg(c, "payment provider already configured")
			return
		}
	}
	config, err := paymentConfigFromRequest(req)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if err := model.CreatePaymentConfig(&config); err != nil {
		common.ApiErrorMsg(c, "failed to create payment config")
		return
	}
	common.ApiSuccess(c, config.PublicView())
}

func UpdatePaymentConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid payment config id")
		return
	}
	existing, err := model.GetPaymentConfigByID(id)
	if err != nil {
		common.ApiErrorMsg(c, "payment config not found")
		return
	}
	var req paymentConfigWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid payment config")
		return
	}
	updates := make(map[string]any)
	publicValues := map[string]any{
		"display_name": req.DisplayName, "icon": req.Icon, "enabled": req.Enabled,
		"sort_order": req.SortOrder, "app_id": req.AppID, "wechat_app_id": req.WechatAppID,
		"wechat_mch_id": req.WechatMchID, "wechat_serial_no": req.WechatSerialNo,
		"wechat_auth_mode": req.WechatAuthMode, "wechat_public_key_id": req.WechatPublicKeyID,
		"gateway_url": req.GatewayURL, "notify_url": req.NotifyURL, "return_url": req.ReturnURL,
	}
	for field, value := range publicValues {
		switch typed := value.(type) {
		case *string:
			if typed != nil {
				updates[field] = *typed
			}
		case *bool:
			if typed != nil {
				updates[field] = *typed
			}
		case *int:
			if typed != nil {
				updates[field] = *typed
			}
		}
	}
	newConfig := *existing
	if req.Provider != nil && *req.Provider != existing.Provider {
		common.ApiErrorMsg(c, "payment provider cannot be changed")
		return
	}
	if req.DisplayName != nil {
		newConfig.DisplayName = *req.DisplayName
	}
	if req.Icon != nil {
		newConfig.Icon = *req.Icon
	}
	if req.Enabled != nil {
		newConfig.Enabled = *req.Enabled
	}
	if req.SortOrder != nil {
		newConfig.SortOrder = *req.SortOrder
	}
	if req.AppID != nil {
		newConfig.AppID = *req.AppID
	}
	if req.WechatAppID != nil {
		newConfig.WechatAppID = *req.WechatAppID
	}
	if req.WechatMchID != nil {
		newConfig.WechatMchID = *req.WechatMchID
	}
	if req.WechatSerialNo != nil {
		newConfig.WechatSerialNo = *req.WechatSerialNo
	}
	if req.WechatAuthMode != nil {
		newConfig.WechatAuthMode = *req.WechatAuthMode
	}
	if req.WechatPublicKeyID != nil {
		newConfig.WechatPublicKeyID = *req.WechatPublicKeyID
	}
	if req.GatewayURL != nil {
		newConfig.GatewayURL = *req.GatewayURL
	}
	if req.NotifyURL != nil {
		newConfig.NotifyURL = *req.NotifyURL
	}
	if req.ReturnURL != nil {
		newConfig.ReturnURL = *req.ReturnURL
	}
	if err := writePaymentConfigSecrets(req, &newConfig); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	newSecrets := map[string]string{
		"app_private_key": newConfig.AppPrivateKey, "alipay_public_key": newConfig.AlipayPublicKey,
		"alipay_app_public_cert": newConfig.AlipayAppPublicCert, "alipay_public_cert": newConfig.AlipayPublicCert,
		"alipay_root_cert": newConfig.AlipayRootCert, "wechat_app_secret": newConfig.WechatAppSecret,
		"wechat_api_key": newConfig.WechatAPIKey, "wechat_private_key": newConfig.WechatPrivateKey,
		"wechat_cert": newConfig.WechatCert, "wechat_public_key": newConfig.WechatPublicKey,
	}
	existingSecrets := map[string]string{
		"app_private_key": existing.AppPrivateKey, "alipay_public_key": existing.AlipayPublicKey,
		"alipay_app_public_cert": existing.AlipayAppPublicCert, "alipay_public_cert": existing.AlipayPublicCert,
		"alipay_root_cert": existing.AlipayRootCert, "wechat_app_secret": existing.WechatAppSecret,
		"wechat_api_key": existing.WechatAPIKey, "wechat_private_key": existing.WechatPrivateKey,
		"wechat_cert": existing.WechatCert, "wechat_public_key": existing.WechatPublicKey,
	}
	for field, value := range newSecrets {
		if value != "" && value != existingSecrets[field] {
			updates[field] = value
		}
	}
	for _, field := range req.ClearSecrets {
		if _, ok := paymentConfigSecretFields[field]; !ok {
			common.ApiErrorMsg(c, "unknown payment secret field")
			return
		}
		updates[field] = ""
		switch field {
		case "app_private_key":
			newConfig.AppPrivateKey = ""
		case "alipay_public_key":
			newConfig.AlipayPublicKey = ""
		case "alipay_app_public_cert":
			newConfig.AlipayAppPublicCert = ""
		case "alipay_public_cert":
			newConfig.AlipayPublicCert = ""
		case "alipay_root_cert":
			newConfig.AlipayRootCert = ""
		case "wechat_app_secret":
			newConfig.WechatAppSecret = ""
		case "wechat_api_key":
			newConfig.WechatAPIKey = ""
		case "wechat_private_key":
			newConfig.WechatPrivateKey = ""
		case "wechat_cert":
			newConfig.WechatCert = ""
		case "wechat_public_key":
			newConfig.WechatPublicKey = ""
		}
	}
	if err := validatePaymentConfigForEnable(&newConfig); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if err := model.UpdatePaymentConfigFields(id, updates); err != nil {
		common.ApiErrorMsg(c, "failed to update payment config")
		return
	}
	updated, err := model.GetPaymentConfigByID(id)
	if err != nil {
		common.ApiErrorMsg(c, "payment config not found")
		return
	}
	common.ApiSuccess(c, updated.PublicView())
}

func GetPaymentConfigs(c *gin.Context) {
	configs, err := model.GetAllPaymentConfigs()
	if err != nil {
		common.ApiErrorMsg(c, "failed to get payment configs")
		return
	}
	views := make([]model.PaymentConfigView, 0, len(configs))
	for i := range configs {
		views = append(views, configs[i].PublicView())
	}
	common.ApiSuccess(c, views)
}

func GetPaymentConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid payment config id")
		return
	}
	config, err := model.GetPaymentConfigByID(id)
	if err != nil {
		common.ApiErrorMsg(c, "payment config not found")
		return
	}
	common.ApiSuccess(c, config.PublicView())
}

func DeletePaymentConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid payment config id")
		return
	}
	if err := model.DeletePaymentConfig(id); err != nil {
		common.ApiErrorMsg(c, "failed to delete payment config")
		return
	}
	common.ApiSuccess(c, nil)
}
