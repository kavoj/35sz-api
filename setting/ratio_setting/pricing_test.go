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
package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests protect the "billing fallback chain" for video generation:
//
//   1. Admin-configured VideoPricing map (per-model, editable)
//   2. Compile-time defaultVideoPricing seed
//   3. videoPricingHardcodedFallback — mirrors the pre-refactor VeoResolutionRatio
//
// If any layer disagrees with the others, admins get surprise pricing.

func TestVideoPricingSeedIncludesVeoAndSeedance(t *testing.T) {
	// Reset the map so we exercise a fresh Seed call.
	videoPricingMap.Clear()
	SeedDefaultVideoPricing()

	// Veo 3.1 must be seeded with a 4K multiplier of 1.5 to match the
	// pre-refactor VeoResolutionRatio behavior for the audio-on variant.
	veo, ok := GetVideoPricing("veo-3.1-generate")
	require.True(t, ok, "veo-3.1-generate must be seeded")
	assert.InDelta(t, 0.40, veo.PricePerSecond, 1e-6)
	assert.InDelta(t, 1.5, veo.ResolutionMultipliers["4k"], 1e-6)

	// Seedance 2.0 base — mirrors the doubao videoPriceTable base entry.
	sd, ok := GetVideoPricing("doubao-seedance-2-0-260128")
	require.True(t, ok, "doubao seedance 2.0 must be seeded")
	assert.InDelta(t, 6.301369, sd.PricePerSecond, 1e-4,
		"seedance base price should equal 46 CNY / 7.3 USDExchangeRate")
}

func TestSeedDefaultVideoPricingDoesNotOverwriteAdminValues(t *testing.T) {
	videoPricingMap.Clear()
	// Simulate an admin edit landing in the map before startup completes.
	adminOverride := VideoPricing{
		PricePerSecond: 999.0,
		ResolutionMultipliers: map[string]float64{
			"4k": 42.0,
		},
	}
	videoPricingMap.Set("veo-3.1-generate", adminOverride)

	SeedDefaultVideoPricing()

	got, ok := GetVideoPricing("veo-3.1-generate")
	require.True(t, ok)
	assert.InDelta(t, 999.0, got.PricePerSecond, 1e-6,
		"seed must not overwrite an admin-configured value")
}

func TestResolutionMultiplierPrefersAdminOverFallback(t *testing.T) {
	videoPricingMap.Clear()
	videoPricingMap.Set("veo-3.1-generate", VideoPricing{
		PricePerSecond: 0.40,
		ResolutionMultipliers: map[string]float64{
			"4k": 3.0, // admin decided to charge 3x instead of the 1.5x default
		},
	})

	// Admin value wins.
	assert.Equal(t, 3.0, ResolutionMultiplierWithFallback("veo-3.1-generate", "4k"))
}

func TestResolutionMultiplierFallbackWhenNoAdminEntry(t *testing.T) {
	videoPricingMap.Clear()
	// No admin entry — the hardcoded fallback for veo-3.1 returns 1.5.
	assert.InDelta(t, 1.5,
		ResolutionMultiplierWithFallback("veo-3.1-generate-preview", "4k"), 1e-6)
	// A non-Veo model with no entry defaults to 1.0 for any resolution.
	assert.Equal(t, 1.0,
		ResolutionMultiplierWithFallback("some-custom-video-model", "4k"))
	// 720p without any entry is always 1.0 (no resolution uplift).
	assert.Equal(t, 1.0,
		ResolutionMultiplierWithFallback("veo-3.1-generate", "720p"))
}

func TestVideoPricingJSONRoundTrip(t *testing.T) {
	videoPricingMap.Clear()
	original := VideoPricing{
		PricePerSecond: 0.5,
		ResolutionMultipliers: map[string]float64{
			"720p":  1.0,
			"1080p": 2.0,
		},
		HasAudioMultiplier: 1.5,
	}
	videoPricingMap.Set("my-model", original)

	blob := VideoPricing2JSONString()
	require.NotEmpty(t, blob)

	videoPricingMap.Clear()
	require.NoError(t, UpdateVideoPricingByJSONString(blob))

	restored, ok := GetVideoPricing("my-model")
	require.True(t, ok)
	assert.InDelta(t, 0.5, restored.PricePerSecond, 1e-9)
	assert.InDelta(t, 2.0, restored.ResolutionMultipliers["1080p"], 1e-9)
	assert.InDelta(t, 1.5, restored.HasAudioMultiplier, 1e-9)
}

func TestImageAndAudioPricingSeeds(t *testing.T) {
	imagePricingMap.Clear()
	SeedDefaultImagePricing()
	dalle, ok := GetImagePricing("dall-e-3")
	require.True(t, ok)
	assert.InDelta(t, 0.04, dalle.PricePerImage, 1e-9)
	assert.InDelta(t, 2.0, dalle.QualityMultipliers["hd"], 1e-9)

	audioInPricingMap.Clear()
	SeedDefaultAudioInPricing()
	whisper, ok := GetAudioInPricing("whisper-1")
	require.True(t, ok)
	assert.InDelta(t, 0.006, whisper.PricePerMinute, 1e-9)

	audioOutPricingMap.Clear()
	SeedDefaultAudioOutPricing()
	tts, ok := GetAudioOutPricing("tts-1")
	require.True(t, ok)
	assert.InDelta(t, 15.0, tts.PricePerMillionChars, 1e-9)
}
