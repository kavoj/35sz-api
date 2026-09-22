package ali

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelListIncludesTokenPlanHappyHorseModels(t *testing.T) {
	for _, model := range []string{
		"happyhorse-1.1-t2v",
		"happyhorse-1.1-i2v",
		"happyhorse-1.1-r2v",
		"happyhorse-1.0-t2v",
		"happyhorse-1.0-i2v",
		"happyhorse-1.0-r2v",
	} {
		require.Contains(t, ModelList, model)
	}
}
