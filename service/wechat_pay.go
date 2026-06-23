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
	if config == nil {
		return fmt.Errorf("wechat config is nil")
	}
	if config.WechatAppID == "" || config.WechatMchID == "" || config.WechatPrivateKey == "" {
		return fmt.Errorf("wechat pay config is incomplete")
	}
	if config.WechatSerialNo == "" && config.WechatCert == "" {
		return fmt.Errorf("wechat merchant certificate or serial number is required")
	}
	if config.WechatAuthMode == "public_key" {
		if config.WechatPublicKeyID == "" || config.WechatPublicKey == "" {
			return fmt.Errorf("wechat pay public key config is incomplete")
		}
		return nil
	}
	if config.WechatAPIKey == "" {
		return fmt.Errorf("wechat pay api v3 key is required")
	}
	return nil
}

func parseWechatMerchantCertSerial(certPEM string) (string, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(certPEM)))
	if block == nil {
		return "", fmt.Errorf("decode wechat merchant certificate failed")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse wechat merchant certificate failed: %w", err)
	}
	return strings.ToUpper(cert.SerialNumber.Text(16)), nil
}

func getWechatMerchantSerialNo(config *model.PaymentConfig) (string, error) {
	if strings.TrimSpace(config.WechatSerialNo) != "" {
		return strings.TrimSpace(config.WechatSerialNo), nil
	}
	if strings.TrimSpace(config.WechatCert) == "" {
		return "", fmt.Errorf("wechat merchant certificate serial number is required")
	}
	return parseWechatMerchantCertSerial(config.WechatCert)
}

func loadWechatPayPublicKey(publicKeyPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(publicKeyPEM)))
	if block == nil {
		return nil, fmt.Errorf("decode wechat pay public key failed")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse wechat pay public key failed: %w", err)
	}
	publicKey, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("wechat pay public key is not RSA")
	}
	return publicKey, nil
}

// loadWechatPrivateKey loads the merchant private key, tolerating common
// corruptions introduced when the PEM is pasted as a single line — namely
// literal "\n"/"\r\n" escape sequences (e.g. copied out of a single-line JSON
// config) and CRLF line endings. It first tries the raw value, then retries
// once with normalized newlines before giving up.
func loadWechatPrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	key, err := utils.LoadPrivateKey(privateKeyPEM)
	if err == nil {
		return key, nil
	}

	normalized := normalizeWechatPEM(privateKeyPEM)
	if normalized == privateKeyPEM {
		return nil, err
	}
	if key, retryErr := utils.LoadPrivateKey(normalized); retryErr == nil {
		return key, nil
	}
	return nil, err
}

// normalizeWechatPEM converts literal escape sequences and CRLF into real
// newlines so a PEM block damaged by single-line copy/paste can be decoded.
func normalizeWechatPEM(s string) string {
	replaced := strings.ReplaceAll(s, "\\r\\n", "\n")
	replaced = strings.ReplaceAll(replaced, "\\n", "\n")
	replaced = strings.ReplaceAll(replaced, "\\r", "\n")
	replaced = strings.ReplaceAll(replaced, "\r\n", "\n")
	replaced = strings.ReplaceAll(replaced, "\r", "\n")
	return replaced
}


func NewWechatPayClient(config *model.PaymentConfig, notifyPath string) (*WechatPayClient, error) {
	if err := validateWechatPayConfig(config); err != nil {
		return nil, err
	}

	privateKey, err := utils.LoadPrivateKeyWithPath(config.WechatPrivateKey)
	if err != nil {
		privateKey, err = loadWechatPrivateKey(config.WechatPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("load wechat private key failed: %w", err)
		}
	}

	serialNo, err := getWechatMerchantSerialNo(config)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	var client *core.Client
	var handler *notify.Handler

	if config.WechatAuthMode == "public_key" {
		if config.WechatPublicKeyID == "" || config.WechatPublicKey == "" {
			return nil, fmt.Errorf("wechat pay public key config is incomplete")
		}
		publicKey, err := loadWechatPayPublicKey(config.WechatPublicKey)
		if err != nil {
			return nil, err
		}
		client, err = core.NewClient(ctx, option.WithWechatPayPublicKeyAuthCipher(config.WechatMchID, serialNo, privateKey, config.WechatPublicKeyID, publicKey))
		if err != nil {
			return nil, fmt.Errorf("init wechat pay public key client failed: %w", err)
		}
		handler = notify.NewNotifyHandler(config.WechatAPIKey, verifiers.NewSHA256WithRSAPubkeyVerifier(config.WechatPublicKeyID, *publicKey))
	} else {
		if err := downloader.MgrInstance().RegisterDownloaderWithPrivateKey(ctx, privateKey, serialNo, config.WechatMchID, config.WechatAPIKey); err != nil {
			return nil, fmt.Errorf("register wechat pay downloader failed: %w", err)
		}
		client, err = core.NewClient(ctx, option.WithWechatPayAutoAuthCipher(config.WechatMchID, serialNo, privateKey, config.WechatAPIKey))
		if err != nil {
			return nil, fmt.Errorf("init wechat pay client failed: %w", err)
		}
		certificateVisitor := downloader.MgrInstance().GetCertificateVisitor(config.WechatMchID)
		handler = notify.NewNotifyHandler(config.WechatAPIKey, verifiers.NewSHA256WithRSAVerifier(certificateVisitor))
	}

	notifyURL := config.NotifyURL
	if notifyURL == "" {
		notifyURL = GetCallbackAddress() + notifyPath
	}

	return &WechatPayClient{client: client, appID: config.WechatAppID, mchID: config.WechatMchID, notifyURL: notifyURL, handler: handler}, nil
}

func (w *WechatPayClient) CreateNativeOrder(ctx context.Context, tradeNo string, description string, amountInFen int64, expireTime time.Time) (string, error) {
	svc := native.NativeApiService{Client: w.client}
	resp, _, err := svc.Prepay(ctx, native.PrepayRequest{
		Appid:       core.String(w.appID),
		Mchid:       core.String(w.mchID),
		Description: core.String(description),
		OutTradeNo:  core.String(tradeNo),
		TimeExpire:  &expireTime,
		NotifyUrl:   core.String(w.notifyURL),
		Amount: &native.Amount{
			Total:    core.Int64(amountInFen),
			Currency: core.String("CNY"),
		},
	})
	if err != nil {
		return "", fmt.Errorf("wechat native prepay failed: %w", err)
	}
	if resp.CodeUrl == nil {
		return "", fmt.Errorf("wechat native prepay response missing code_url")
	}
	return *resp.CodeUrl, nil
}

func (w *WechatPayClient) CreateH5Order(ctx context.Context, tradeNo string, description string, amountInFen int64, expireTime time.Time, payerClientIP string) (string, error) {
	svc := h5.H5ApiService{Client: w.client}
	resp, _, err := svc.Prepay(ctx, h5.PrepayRequest{
		Appid:       core.String(w.appID),
		Mchid:       core.String(w.mchID),
		Description: core.String(description),
		OutTradeNo:  core.String(tradeNo),
		TimeExpire:  &expireTime,
		NotifyUrl:   core.String(w.notifyURL),
		Amount: &h5.Amount{
			Total:    core.Int64(amountInFen),
			Currency: core.String("CNY"),
		},
		SceneInfo: &h5.SceneInfo{
			PayerClientIp: core.String(payerClientIP),
			H5Info: &h5.H5Info{
				Type: core.String("Wap"),
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("wechat h5 prepay failed: %w", err)
	}
	if resp.H5Url == nil {
		return "", fmt.Errorf("wechat h5 prepay response missing h5_url")
	}
	return *resp.H5Url, nil
}

func (w *WechatPayClient) ParseNotifyRequest(ctx context.Context, request *http.Request, content interface{}) (*notify.Request, error) {
	return w.handler.ParseNotifyRequest(ctx, request, content)
}
