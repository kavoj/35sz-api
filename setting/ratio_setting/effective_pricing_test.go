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

// PR-5 wires GetModelPrice to consult the 4 structured pricing tables before
// falling through to the legacy ModelPrice map. These tests pin down that
// ordering and the fallback contract, since any regression here would
// silently misbill.

// clearAllStructuredPricing empties the 4 maps between test cases so each
// test controls what's present.
func clearAllStructuredPricing() {
	imagePricingMap.Clear()
	videoPricingMap.Clear()
	audioInPricingMap.Clear()
	audioOutPricingMap.Clear()
}

func TestEffectivePerRequestPriceEmptyMapsReturnFalse(t *testing.T) {
	clearAllStructuredPricing()
	_, ok := EffectivePerRequestPrice("some-model")
	assert.False(t, ok, "no structured entry should return (0, false)")
}

func TestEffectivePerRequestPriceImageGen(t *testing.T) {
	clearAllStructuredPricing()
	imagePricingMap.Set("flux-1.1-pro", ImagePricing{PricePerImage: 0.04})

	price, ok := EffectivePerRequestPrice("flux-1.1-pro")
	require.True(t, ok)
	assert.InDelta(t, 0.04, price, 1e-9)
}

func TestEffectivePerRequestPriceVideoGen(t *testing.T) {
	clearAllStructuredPricing()
	videoPricingMap.Set("doubao-seedance-2-0", VideoPricing{
		PricePerSecond: 6.301369,
	})

	price, ok := EffectivePerRequestPrice("doubao-seedance-2-0")
	require.True(t, ok)
	assert.InDelta(t, 6.301369, price, 1e-6)
}

func TestEffectivePerRequestPriceAudioIn(t *testing.T) {
	clearAllStructuredPricing()
	audioInPricingMap.Set("whisper-1", AudioInPricing{PricePerMinute: 0.006})

	price, ok := EffectivePerRequestPrice("whisper-1")
	require.True(t, ok)
	assert.InDelta(t, 0.006, price, 1e-9)
}

func TestEffectivePerRequestPriceAudioOut(t *testing.T) {
	clearAllStructuredPricing()
	audioOutPricingMap.Set("tts-1", AudioOutPricing{PricePerMillionChars: 15.0})

	price, ok := EffectivePerRequestPrice("tts-1")
	require.True(t, ok)
	assert.InDelta(t, 15.0, price, 1e-9)
}

func TestEffectivePerRequestPriceZeroValueTreatedAsUnset(t *testing.T) {
	// If admin saves an entry with price=0 (unusual but possible during
	// mid-edit state), we treat it as "not configured" so the fallback to
	// legacy ModelPrice kicks in. This prevents an accidental 0 from
	// silently becoming a free model.
	clearAllStructuredPricing()
	imagePricingMap.Set("half-configured", ImagePricing{PricePerImage: 0})

	_, ok := EffectivePerRequestPrice("half-configured")
	assert.False(t, ok, "price=0 should be treated as unset")
}

// TestGetModelPricePrefersStructuredOverLegacy is the core PR-5 contract:
// if a model has both a legacy `ModelPrice` entry AND a structured entry,
// the structured one wins. This lets admins migrate a model from legacy to
// structured pricing without deleting the old row first.
func TestGetModelPricePrefersStructuredOverLegacy(t *testing.T) {
	clearAllStructuredPricing()
	modelPriceMap.Clear()

	modelPriceMap.Set("hybrid-model", 0.02)
	videoPricingMap.Set("hybrid-model", VideoPricing{PricePerSecond: 0.05})

	price, ok := GetModelPrice("hybrid-model", false)
	require.True(t, ok)
	assert.InDelta(t, 0.05, price, 1e-9,
		"structured video entry must win over legacy ModelPrice")
}

// TestGetModelPriceFallsBackToLegacy covers the zero-behavior-change
// guarantee: token-billed models (chat / embedding) with only legacy config
// still work exactly as before PR-5.
func TestGetModelPriceFallsBackToLegacy(t *testing.T) {
	clearAllStructuredPricing()
	modelPriceMap.Clear()
	modelPriceMap.Set("legacy-only", 0.01)

	price, ok := GetModelPrice("legacy-only", false)
	require.True(t, ok)
	assert.InDelta(t, 0.01, price, 1e-9)
}

// TestGetModelPriceUnknownModelReturnsFalse guards the "model not priced"
// error path in ModelPriceHelper — used to render the admin-facing error
// message when a model is unconfigured across both structured and legacy.
func TestGetModelPriceUnknownModelReturnsFalse(t *testing.T) {
	clearAllStructuredPricing()
	modelPriceMap.Clear()

	_, ok := GetModelPrice("never-priced-model", false)
	assert.False(t, ok)
}
