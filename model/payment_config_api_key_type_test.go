package model

import (
	"reflect"
	"strings"
	"testing"
)

func TestPaymentConfigWechatAPIKeyColumnUsesText(t *testing.T) {
	typeOf := reflect.TypeOf(PaymentConfig{})
	fieldInfo, ok := typeOf.FieldByName("WechatAPIKey")
	if !ok {
		t.Fatal("WechatAPIKey field not found")
	}
	if tag := fieldInfo.Tag.Get("gorm"); !strings.Contains(tag, "type:text") {
		t.Fatalf("WechatAPIKey gorm tag = %q, want type:text", tag)
	}
}
