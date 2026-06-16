package service

import "testing"

const testWechatPayPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA7EwX9SwoUpqL/KRJaOAt
96LVPLMTayd/88wAYmFc0zls/ZpvSL7ukTyjg7nzzFszOtD+wb5nMo12+ttjmQv7
XMBzEJMye43Eul9uROQKKJNhr51Gbvyiysb9AEQe7LN/l1716NPRQVdNyJTb6/W0
7xn8ZbtqH+2DG56e716VRSUKswllQZ8ezlaJL6GtddYr+1x4BRwPFeu7furY7q/Y
UYikAUnabBLkRZPxWIwsEyhkP1A63mpm1WJhxEEVn1LmZkVIJcHobFXxWNQRaFU9
q3kMyfE7BiJG4IfRtWxdeywx17yTclSe1/YUr+evgsGwEXYCPOBwnOq+yldhJAmD
ZQIDAQAB
-----END PUBLIC KEY-----`

func TestLoadWechatPayPublicKeyParsesPEM(t *testing.T) {
	key, err := loadWechatPayPublicKey(testWechatPayPublicKey)
	if err != nil {
		t.Fatalf("loadWechatPayPublicKey returned error: %v", err)
	}
	if key == nil || key.N == nil || key.E == 0 {
		t.Fatalf("parsed public key is invalid: %+v", key)
	}
}
