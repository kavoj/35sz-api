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
package doubao

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// PR-2 adds an admin-configurable path for doubao seedance resolution
// multipliers. Only the hasVideo=false axis (pure text→video) is wired for
// now — the hasVideo=true dimension stays on the hardcoded videoPriceTable
// until PR-5 generalizes the setting struct.

func TestGetVideoInputRatioPrefersAdminSettingWhenNoVideoInput(t *testing.T) {
	ratio_setting.ClearVideoPricingForTest()
	_ = ratio_setting.SetVideoPricingForTest("doubao-seedance-2-0-260128", ratio_setting.VideoPricing{
		PricePerSecond: 6.301369,
		ResolutionMultipliers: map[string]float64{
			"1080p": 3.0, // admin wants 3x for 1080p instead of the 51/46 default
		},
	})

	ratio, ok := GetVideoInputRatio("doubao-seedance-2-0-260128", "1080p", false)
	assert.True(t, ok)
	assert.InDelta(t, 3.0, ratio, 1e-9, "admin value must win when hasVideo=false")
}

func TestGetVideoInputRatioFallsBackWhenVideoInput(t *testing.T) {
	// Admin-configured overrides do NOT apply when the request carries video
	// input — seedance charges a distinct rate for that, and the admin UI
	// doesn't expose that axis yet. PR-5 will generalize.
	ratio_setting.ClearVideoPricingForTest()
	_ = ratio_setting.SetVideoPricingForTest("doubao-seedance-2-0-260128", ratio_setting.VideoPricing{
		PricePerSecond: 6.301369,
		ResolutionMultipliers: map[string]float64{
			"1080p": 3.0,
		},
	})

	// hasVideo=true → walks the hardcoded path: 31/46 = 0.673913
	ratio, ok := GetVideoInputRatio("doubao-seedance-2-0-260128", "1080p", true)
	assert.True(t, ok)
	assert.InDelta(t, 31.0/46.0, ratio, 1e-6,
		"admin path must be skipped when hasVideo=true (until PR-5 wires it)")
}

func TestGetVideoInputRatioFallsBackToHardcodedWhenNoAdminEntry(t *testing.T) {
	ratio_setting.ClearVideoPricingForTest()

	// Base tier: 480p/720p no video → base price, ratio = 1.0
	ratio, ok := GetVideoInputRatio("doubao-seedance-2-0-260128", "720p", false)
	assert.True(t, ok)
	assert.Equal(t, 1.0, ratio)

	// 1080p no video → 51/46 ≈ 1.1087
	ratio, ok = GetVideoInputRatio("doubao-seedance-2-0-260128", "1080p", false)
	assert.True(t, ok)
	assert.InDelta(t, 51.0/46.0, ratio, 1e-6)

	// Unknown model → not configured
	_, ok = GetVideoInputRatio("some-other-model", "720p", false)
	assert.False(t, ok)
}
