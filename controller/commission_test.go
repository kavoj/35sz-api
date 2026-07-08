package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaskUsername(t *testing.T) {
	require.Equal(t, "张*", maskUsername("张三"))
	require.Equal(t, "a*****", maskUsername("abcdef"))
	require.Equal(t, "a", maskUsername("a"))
	require.Equal(t, "", maskUsername(""))
}

func TestMaskEmail(t *testing.T) {
	require.Equal(t, "t***@example.com", maskEmail("test@example.com"))
	require.Equal(t, "a", maskEmail("a"))       // no @
	require.Equal(t, "@x", maskEmail("@x"))     // no local part
}
