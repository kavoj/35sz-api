package controller

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func getDecryptedPaymentConfig(provider string) (*model.PaymentConfig, error) {
	config, err := model.GetEnabledPaymentConfigByProvider(provider)
	if err != nil {
		return nil, err
	}
	secretFields := []*string{
		&config.AppPrivateKey, &config.AlipayPublicKey, &config.AlipayAppPublicCert,
		&config.AlipayPublicCert, &config.AlipayRootCert, &config.WechatAppSecret,
		&config.WechatAPIKey, &config.WechatPrivateKey, &config.WechatCert,
		&config.WechatPublicKey,
	}
	for _, field := range secretFields {
		if *field == "" {
			continue
		}
		plaintext, err := common.DecryptPaymentSecret(*field)
		if err != nil {
			return nil, errors.New("invalid encrypted payment configuration")
		}
		*field = plaintext
	}
	return config, nil
}
