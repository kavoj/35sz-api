package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestNewAlipayPayClientRejectsIncompleteConfig(t *testing.T) {
	_, err := NewAlipayPayClient(&model.PaymentConfig{AppID: "app-1"}, "/api/alipay/notify", "")
	require.Error(t, err)

	_, err = NewAlipayPayClient(&model.PaymentConfig{AppID: "app-1", AppPrivateKey: "private-key"}, "/api/alipay/notify", "")
	require.Error(t, err)
}
