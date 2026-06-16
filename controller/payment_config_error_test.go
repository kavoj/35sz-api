package controller

import (
	"errors"
	"strings"
	"testing"
)

func TestPaymentConfigCreateErrorMessageIncludesCause(t *testing.T) {
	msg := paymentConfigCreateErrorMessage(errors.New("table payment_configs has no column named wechat_app_secret"))
	if !strings.Contains(msg, "wechat_app_secret") {
		t.Fatalf("message = %q, want underlying cause", msg)
	}
}
