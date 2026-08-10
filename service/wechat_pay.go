package service

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

type WechatPayClient struct {
	client    *core.Client
	appID     string
	mchID     string
	notifyURL string
	handler   *notify.Handler
}

func validateWechatPayConfig(config *model.PaymentConfig) error {
	if config == nil || config.WechatAppID == "" || config.WechatMchID == "" || config.WechatPrivateKey == "" {
		return fmt.Errorf("wechat pay config is incomplete")
	}
	if config.WechatSerialNo == "" && config.WechatCert == "" {
		return fmt.Errorf("wechat merchant serial or certificate is required")
	}
	if config.WechatAuthMode == "public_key" {
		if config.WechatPublicKeyID == "" || config.WechatPublicKey == "" {
			return fmt.Errorf("wechat public key configuration is incomplete")
		}
		return nil
	}
	if config.WechatAPIKey == "" {
		return fmt.Errorf("wechat api v3 key is required")
	}
	return nil
}

func normalizeWechatPEM(value string) string {
	value = strings.ReplaceAll(value, "\\r\\n", "\n")
	value = strings.ReplaceAll(value, "\\n", "\n")
	value = strings.ReplaceAll(value, "\\r", "\n")
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\r", "\n")
}

func loadWechatPrivateKey(value string) (*rsa.PrivateKey, error) {
	if key, err := utils.LoadPrivateKey(value); err == nil {
		return key, nil
	}
	return utils.LoadPrivateKey(normalizeWechatPEM(value))
}

func wechatMerchantSerial(config *model.PaymentConfig) (string, error) {
	if strings.TrimSpace(config.WechatSerialNo) != "" {
		return strings.TrimSpace(config.WechatSerialNo), nil
	}
	block, _ := pem.Decode([]byte(strings.TrimSpace(normalizeWechatPEM(config.WechatCert))))
	if block == nil {
		return "", fmt.Errorf("decode wechat merchant certificate failed")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse wechat merchant certificate failed: %w", err)
	}
	return strings.ToUpper(cert.SerialNumber.Text(16)), nil
}

func loadWechatPublicKey(value string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(normalizeWechatPEM(value))))
	if block == nil {
		return nil, fmt.Errorf("decode wechat public key failed")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse wechat public key failed: %w", err)
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("wechat public key is not RSA")
	}
	return key, nil
}

func NewWechatPayClient(config *model.PaymentConfig, notifyPath string) (*WechatPayClient, error) {
	if err := validateWechatPayConfig(config); err != nil {
		return nil, err
	}
	privateKey, err := loadWechatPrivateKey(config.WechatPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("load wechat private key failed: %w", err)
	}
	serial, err := wechatMerchantSerial(config)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var client *core.Client
	var handler *notify.Handler
	if config.WechatAuthMode == "public_key" {
		publicKey, err := loadWechatPublicKey(config.WechatPublicKey)
		if err != nil {
			return nil, err
		}
		client, err = core.NewClient(ctx, option.WithWechatPayPublicKeyAuthCipher(config.WechatMchID, serial, privateKey, config.WechatPublicKeyID, publicKey))
		if err != nil {
			return nil, fmt.Errorf("init wechat public key client: %w", err)
		}
		handler = notify.NewNotifyHandler(config.WechatAPIKey, verifiers.NewSHA256WithRSAPubkeyVerifier(config.WechatPublicKeyID, *publicKey))
	} else {
		if err := downloader.MgrInstance().RegisterDownloaderWithPrivateKey(ctx, privateKey, serial, config.WechatMchID, config.WechatAPIKey); err != nil {
			return nil, fmt.Errorf("register wechat certificate downloader: %w", err)
		}
		client, err = core.NewClient(ctx, option.WithWechatPayAutoAuthCipher(config.WechatMchID, serial, privateKey, config.WechatAPIKey))
		if err != nil {
			return nil, fmt.Errorf("init wechat certificate client: %w", err)
		}
		handler = notify.NewNotifyHandler(config.WechatAPIKey, verifiers.NewSHA256WithRSAVerifier(downloader.MgrInstance().GetCertificateVisitor(config.WechatMchID)))
	}
	notifyURL := config.NotifyURL
	if notifyURL == "" {
		notifyURL = GetCallbackAddress() + notifyPath
	}
	return &WechatPayClient{client: client, appID: config.WechatAppID, mchID: config.WechatMchID, notifyURL: notifyURL, handler: handler}, nil
}

func (client *WechatPayClient) CreateNativeOrder(ctx context.Context, tradeNo string, description string, amountFen int64, expireTime time.Time) (string, error) {
	resp, _, err := (&native.NativeApiService{Client: client.client}).Prepay(ctx, native.PrepayRequest{Appid: core.String(client.appID), Mchid: core.String(client.mchID), Description: core.String(description), OutTradeNo: core.String(tradeNo), TimeExpire: &expireTime, NotifyUrl: core.String(client.notifyURL), Amount: &native.Amount{Total: core.Int64(amountFen), Currency: core.String("CNY")}})
	if err != nil {
		return "", fmt.Errorf("wechat native prepay failed: %w", err)
	}
	if resp.CodeUrl == nil {
		return "", fmt.Errorf("wechat native response missing code_url")
	}
	return *resp.CodeUrl, nil
}

func (client *WechatPayClient) CreateH5Order(ctx context.Context, tradeNo string, description string, amountFen int64, expireTime time.Time, payerIP string) (string, error) {
	resp, _, err := (&h5.H5ApiService{Client: client.client}).Prepay(ctx, h5.PrepayRequest{Appid: core.String(client.appID), Mchid: core.String(client.mchID), Description: core.String(description), OutTradeNo: core.String(tradeNo), TimeExpire: &expireTime, NotifyUrl: core.String(client.notifyURL), Amount: &h5.Amount{Total: core.Int64(amountFen), Currency: core.String("CNY")}, SceneInfo: &h5.SceneInfo{PayerClientIp: core.String(payerIP), H5Info: &h5.H5Info{Type: core.String("Wap")}}})
	if err != nil {
		return "", fmt.Errorf("wechat h5 prepay failed: %w", err)
	}
	if resp.H5Url == nil {
		return "", fmt.Errorf("wechat h5 response missing h5_url")
	}
	return *resp.H5Url, nil
}

func (client *WechatPayClient) ParseNotifyRequest(ctx context.Context, request *http.Request, content any) (*notify.Request, error) {
	return client.handler.ParseNotifyRequest(ctx, request, content)
}

func YuanToFen(amount float64) int64 { return int64(amount*100 + 0.5) }
