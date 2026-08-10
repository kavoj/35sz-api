package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	PaymentProviderAlipay = "alipay"
	PaymentProviderWechat = "wechat"
)

type PaymentConfig struct {
	ID          int    `json:"id" gorm:"primaryKey"`
	Provider    string `json:"provider" gorm:"size:32;not null;uniqueIndex"`
	DisplayName string `json:"display_name" gorm:"size:100"`
	Icon        string `json:"icon" gorm:"size:100"`
	Enabled     bool   `json:"enabled"`
	SortOrder   int    `json:"sort_order"`

	AppID               string `json:"app_id" gorm:"size:255"`
	AppPrivateKey       string `json:"-" gorm:"type:text"`
	AlipayPublicKey     string `json:"-" gorm:"type:text"`
	AlipayAppPublicCert string `json:"-" gorm:"type:text"`
	AlipayPublicCert    string `json:"-" gorm:"type:text"`
	AlipayRootCert      string `json:"-" gorm:"type:text"`

	WechatAppID       string `json:"wechat_app_id" gorm:"size:255"`
	WechatMchID       string `json:"wechat_mch_id" gorm:"size:255"`
	WechatSerialNo    string `json:"wechat_serial_no" gorm:"size:255"`
	WechatAuthMode    string `json:"wechat_auth_mode" gorm:"size:32"`
	WechatPublicKeyID string `json:"wechat_public_key_id" gorm:"size:255"`
	WechatAppSecret   string `json:"-" gorm:"type:text"`
	WechatAPIKey      string `json:"-" gorm:"type:text"`
	WechatPrivateKey  string `json:"-" gorm:"type:text"`
	WechatCert        string `json:"-" gorm:"type:text"`
	WechatPublicKey   string `json:"-" gorm:"type:text"`

	GatewayURL string `json:"gateway_url" gorm:"size:500"`
	NotifyURL  string `json:"notify_url" gorm:"size:500"`
	ReturnURL  string `json:"return_url" gorm:"size:500"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type PaymentConfigView struct {
	ID          int    `json:"id"`
	Provider    string `json:"provider"`
	DisplayName string `json:"display_name"`
	Icon        string `json:"icon"`
	Enabled     bool   `json:"enabled"`
	SortOrder   int    `json:"sort_order"`
	AppID       string `json:"app_id"`

	WechatAppID       string `json:"wechat_app_id"`
	WechatMchID       string `json:"wechat_mch_id"`
	WechatSerialNo    string `json:"wechat_serial_no"`
	WechatAuthMode    string `json:"wechat_auth_mode"`
	WechatPublicKeyID string `json:"wechat_public_key_id"`
	GatewayURL        string `json:"gateway_url"`
	NotifyURL         string `json:"notify_url"`
	ReturnURL         string `json:"return_url"`
	CreatedAt         int64  `json:"created_at"`
	UpdatedAt         int64  `json:"updated_at"`

	AppPrivateKeyConfigured      bool   `json:"app_private_key_configured"`
	AlipayPublicKeyConfigured    bool   `json:"alipay_public_key_configured"`
	AlipayCertificatesConfigured bool   `json:"alipay_certificates_configured"`
	WechatAppSecretConfigured    bool   `json:"wechat_app_secret_configured"`
	WechatAPIKeyConfigured       bool   `json:"wechat_api_key_configured"`
	WechatPrivateKeyConfigured   bool   `json:"wechat_private_key_configured"`
	WechatCertConfigured         bool   `json:"wechat_cert_configured"`
	WechatPublicKeyConfigured    bool   `json:"wechat_public_key_configured"`
	AppPrivateKey                string `json:"-"`
}

func (config *PaymentConfig) PublicView() PaymentConfigView {
	if config == nil {
		return PaymentConfigView{}
	}
	return PaymentConfigView{
		ID: config.ID, Provider: config.Provider, DisplayName: config.DisplayName,
		Icon: config.Icon, Enabled: config.Enabled, SortOrder: config.SortOrder,
		AppID: config.AppID, WechatAppID: config.WechatAppID, WechatMchID: config.WechatMchID,
		WechatSerialNo: config.WechatSerialNo, WechatAuthMode: config.WechatAuthMode,
		WechatPublicKeyID: config.WechatPublicKeyID, GatewayURL: config.GatewayURL,
		NotifyURL: config.NotifyURL, ReturnURL: config.ReturnURL,
		CreatedAt: config.CreatedAt, UpdatedAt: config.UpdatedAt,
		AppPrivateKeyConfigured:      config.AppPrivateKey != "",
		AlipayPublicKeyConfigured:    config.AlipayPublicKey != "",
		AlipayCertificatesConfigured: config.AlipayAppPublicCert != "" && config.AlipayPublicCert != "" && config.AlipayRootCert != "",
		WechatAppSecretConfigured:    config.WechatAppSecret != "",
		WechatAPIKeyConfigured:       config.WechatAPIKey != "",
		WechatPrivateKeyConfigured:   config.WechatPrivateKey != "",
		WechatCertConfigured:         config.WechatCert != "",
		WechatPublicKeyConfigured:    config.WechatPublicKey != "",
	}
}

func IsSupportedPaymentConfigProvider(provider string) bool {
	return provider == PaymentProviderAlipay || provider == PaymentProviderWechat
}

func CreatePaymentConfig(config *PaymentConfig) error {
	if config == nil || !IsSupportedPaymentConfigProvider(config.Provider) {
		return errors.New("unsupported payment provider")
	}
	config.Provider = strings.TrimSpace(config.Provider)
	now := common.GetTimestamp()
	config.CreatedAt = now
	config.UpdatedAt = now
	return DB.Create(config).Error
}

func GetPaymentConfigByID(id int) (*PaymentConfig, error) {
	var config PaymentConfig
	if err := DB.First(&config, id).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func GetPaymentConfigByProvider(provider string) (*PaymentConfig, error) {
	var config PaymentConfig
	if err := DB.Where("provider = ?", provider).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func GetEnabledPaymentConfigByProvider(provider string) (*PaymentConfig, error) {
	var config PaymentConfig
	if err := DB.Where("provider = ? AND enabled = ?", provider, true).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func GetAllPaymentConfigs() ([]PaymentConfig, error) {
	var configs []PaymentConfig
	return configs, DB.Order("sort_order ASC, id ASC").Find(&configs).Error
}

var paymentConfigUpdateFields = map[string]struct{}{
	"display_name": {}, "icon": {}, "enabled": {}, "sort_order": {}, "app_id": {},
	"app_private_key": {}, "alipay_public_key": {}, "alipay_app_public_cert": {},
	"alipay_public_cert": {}, "alipay_root_cert": {}, "wechat_app_id": {},
	"wechat_mch_id": {}, "wechat_serial_no": {}, "wechat_auth_mode": {},
	"wechat_public_key_id": {}, "wechat_app_secret": {}, "wechat_api_key": {},
	"wechat_private_key": {}, "wechat_cert": {}, "wechat_public_key": {},
	"gateway_url": {}, "notify_url": {}, "return_url": {},
}

func UpdatePaymentConfigFields(id int, updates map[string]any) error {
	if id <= 0 {
		return gorm.ErrRecordNotFound
	}
	filtered := make(map[string]any, len(updates)+1)
	for field, value := range updates {
		if _, ok := paymentConfigUpdateFields[field]; ok {
			filtered[field] = value
		}
	}
	filtered["updated_at"] = common.GetTimestamp()
	result := DB.Model(&PaymentConfig{}).Where("id = ?", id).Updates(filtered)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func DeletePaymentConfig(id int) error {
	return DB.Delete(&PaymentConfig{}, id).Error
}
