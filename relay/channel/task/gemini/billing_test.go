/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package gemini

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// PR-2 wires VeoResolutionRatio to consult admin-configurable
// ratio_setting.VideoPricing first, falling back to the hardcoded switch
// for backward compat. These tests pin down that order.

func TestVeoResolutionRatioPrefersAdminSetting(t *testing.T) {
	// Reset the map so we control the state deterministically.
	ratio_setting.ClearVideoPricingForTest()

	// Admin says "veo-3.1-generate 4K should cost 5x" — override the
	// hardcoded 1.5x default.
	require := assert.New(t)
	require.NoError(ratio_setting.SetVideoPricingForTest("veo-3.1-generate", ratio_setting.VideoPricing{
		PricePerSecond: 0.40,
		ResolutionMultipliers: map[string]float64{
			"4k": 5.0,
		},
	}))

	assert.InDelta(t, 5.0, VeoResolutionRatio("veo-3.1-generate", "4k"), 1e-9,
		"admin-configured multiplier must win over hardcoded 1.5x")
}

func TestVeoResolutionRatioFallsBackToHardcoded(t *testing.T) {
	// Clear the map so no admin value exists.
	ratio_setting.ClearVideoPricingForTest()

	// Hardcoded path: veo-3.1-generate at 4K → 1.5x.
	assert.InDelta(t, 1.5, VeoResolutionRatio("veo-3.1-generate", "4k"), 1e-9)
	// veo-3.1-fast-generate at 4K → 2.333333.
	assert.InDelta(t, 2.333333, VeoResolutionRatio("veo-3.1-fast-generate", "4k"), 1e-6)
	// veo-3.0-generate at 4K → 1.0 (does not support 4K).
	assert.Equal(t, 1.0, VeoResolutionRatio("veo-3.0-generate", "4k"))
	// Non-4K → 1.0 regardless of model.
	assert.Equal(t, 1.0, VeoResolutionRatio("veo-3.1-generate", "720p"))
	assert.Equal(t, 1.0, VeoResolutionRatio("veo-3.1-generate", "1080p"))
}
