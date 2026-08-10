package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentSecretEncryptionIsVersionedAndAuthenticated(t *testing.T) {
	previous := CryptoSecret
	CryptoSecret = "payment-test-root-secret"
	t.Cleanup(func() { CryptoSecret = previous })

	ciphertext, err := EncryptPaymentSecret("merchant-private-key")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(ciphertext, paymentSecretPrefix))
	assert.NotContains(t, ciphertext, "merchant-private-key")

	plaintext, err := DecryptPaymentSecret(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, "merchant-private-key", plaintext)

	tampered := ciphertext[:len(ciphertext)-1] + "A"
	_, err = DecryptPaymentSecret(tampered)
	require.Error(t, err)
}

func TestPaymentSecretDecryptionRejectsUnversionedValues(t *testing.T) {
	_, err := DecryptPaymentSecret("plaintext-or-legacy-value")
	require.Error(t, err)
}
