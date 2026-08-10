package service

import (
	"context"
	"fmt"
	"net/url"

	"github.com/QuantumNous/new-api/model"
	"github.com/smartwalle/alipay/v3"
)

type AlipayPayClient struct {
	client    *alipay.Client
	notifyURL string
	returnURL string
}

func NewAlipayPayClient(config *model.PaymentConfig, notifyPath string, returnURL string) (*AlipayPayClient, error) {
	if config == nil || config.AppID == "" || config.AppPrivateKey == "" {
		return nil, fmt.Errorf("alipay app id or private key not configured")
	}
	var opts []alipay.OptionFunc
	if config.GatewayURL != "" {
		opts = append(opts, alipay.WithProductionGateway(config.GatewayURL))
	}
	client, err := alipay.New(config.AppID, config.AppPrivateKey, true, opts...)
	if err != nil {
		return nil, fmt.Errorf("init alipay client failed: %w", err)
	}
	if config.AlipayAppPublicCert != "" || config.AlipayPublicCert != "" || config.AlipayRootCert != "" {
		if config.AlipayAppPublicCert == "" || config.AlipayPublicCert == "" || config.AlipayRootCert == "" {
			return nil, fmt.Errorf("alipay certificate configuration is incomplete")
		}
		if err := client.LoadAppCertPublicKey(config.AlipayAppPublicCert); err != nil {
			return nil, fmt.Errorf("load alipay app cert: %w", err)
		}
		if err := client.LoadAlipayCertPublicKey(config.AlipayPublicCert); err != nil {
			return nil, fmt.Errorf("load alipay public cert: %w", err)
		}
		if err := client.LoadAliPayRootCert(config.AlipayRootCert); err != nil {
			return nil, fmt.Errorf("load alipay root cert: %w", err)
		}
	} else if config.AlipayPublicKey != "" {
		if err := client.LoadAliPayPublicKey(config.AlipayPublicKey); err != nil {
			return nil, fmt.Errorf("load alipay public key: %w", err)
		}
	} else {
		return nil, fmt.Errorf("alipay public key or certificates not configured")
	}
	notifyURL := config.NotifyURL
	if notifyURL == "" {
		notifyURL = GetCallbackAddress() + notifyPath
	}
	if returnURL == "" {
		returnURL = config.ReturnURL
	}
	return &AlipayPayClient{client: client, notifyURL: notifyURL, returnURL: returnURL}, nil
}

func (client *AlipayPayClient) CreatePagePay(tradeNo string, subject string, totalAmount string, expireTime string) (string, error) {
	var p alipay.TradePagePay
	p.NotifyURL = client.notifyURL
	p.ReturnURL = client.returnURL
	p.Subject = subject
	p.OutTradeNo = tradeNo
	p.TotalAmount = totalAmount
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"
	p.TimeExpire = expireTime
	result, err := client.client.TradePagePay(p)
	if err != nil {
		return "", fmt.Errorf("alipay trade page pay failed: %w", err)
	}
	return result.String(), nil
}

func (client *AlipayPayClient) CreateWAPPay(tradeNo string, subject string, totalAmount string, expireTime string, quitURL string) (string, error) {
	var p alipay.TradeWapPay
	p.NotifyURL = client.notifyURL
	p.ReturnURL = client.returnURL
	p.Subject = subject
	p.OutTradeNo = tradeNo
	p.TotalAmount = totalAmount
	p.ProductCode = "QUICK_WAP_WAY"
	p.TimeExpire = expireTime
	p.QuitURL = quitURL
	result, err := client.client.TradeWapPay(p)
	if err != nil {
		return "", fmt.Errorf("alipay trade wap pay failed: %w", err)
	}
	return result.String(), nil
}

func (client *AlipayPayClient) VerifyNotification(ctx context.Context, params url.Values) error {
	return client.client.VerifySign(ctx, params)
}
